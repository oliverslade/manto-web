using System.Text.Json;
using Manto.Web.Configuration;
using Manto.Web.Services;
using Microsoft.Extensions.Options;

namespace Manto.Web.Extensions;

public static class WebApplicationExtensions
{
    public static WebApplication ConfigureMiddleware(this WebApplication app)
    {
        var logger = app.Services.GetRequiredService<ILogger<Program>>();
        var settings = app.Services.GetRequiredService<IOptions<ApplicationSettings>>().Value;
        
        logger.LogInformation("Manto starting on port {Port} ({Environment})", 
            settings.Server.Port, app.Environment.EnvironmentName);

        app.UseSecurityHeaders();

        if (!app.Environment.IsDevelopment())
        {
            if (settings.Security.EnableHsts)
            {
                app.UseHsts();
            }
        }

        app.UseHttpsRedirection();
        app.UseOutputCache();
        app.UseDefaultFiles();
        app.UseStaticFiles();

        return app;
    }

    public static WebApplication MapApiEndpoints(this WebApplication app)
    {
        app.MapGet("/config.js", (IOptions<ApplicationSettings> options) =>
        {
            var settings = options.Value;
            
            var config = new
            {
                providers = settings.Features.SupportedProviders.Select(p => new
                {
                    name = p.Name,
                    displayName = p.DisplayName,
                    apiEndpoint = p.ApiEndpoint,
                    apiVersion = p.ApiVersion
                }).ToArray(),
                api = new
                {
                    anthropicKeyPrefix = settings.Features.Api.AnthropicKeyPrefix,
                    preferredModelId = settings.Features.Api.PreferredModelId,
                    endpoints = new
                    {
                        models = settings.Features.Api.Endpoints.Models,
                        messages = settings.Features.Api.Endpoints.Messages
                    }
                },
                validation = new
                {
                    maxMessageLength = settings.Features.Validation.MaxMessageLength,
                    minApiKeyLength = settings.Features.Validation.MinApiKeyLength
                },
                models = new
                {
                    maxTokens = settings.Features.Models.MaxTokens,
                    temperature = settings.Features.Models.Temperature,
                    systemMessage = settings.Features.Models.SystemMessage
                },
                version = "1.0.0"
            };

            var jsonConfig = JsonSerializer.Serialize(config, new JsonSerializerOptions
            {
                PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
                WriteIndented = false
            });

            var configScript = $"window.MantoConfig = {jsonConfig};";
            return Results.Content(configScript, "application/javascript");
        }).CacheOutput(policy => policy.Expire(TimeSpan.FromMinutes(5)));

        app.MapGet("/api/models", async (HttpContext context, IAnthropicApiService anthropicService) =>
        {
            var requestId = BaseAnthropicService.GenerateRequestId();
            
            if (!context.Request.Headers.TryGetValue("x-api-key", out var apiKeyValues) || 
                string.IsNullOrEmpty(apiKeyValues.FirstOrDefault()))
            {
                return Results.BadRequest(new { error = "API key required" });
            }

            var apiKey = apiKeyValues.First()!;
            
            try
            {
                var result = await anthropicService.GetModelsAsync(apiKey, requestId);
                
                if (!result.IsSuccess)
                {
                    return Results.BadRequest(new { error = result.ErrorMessage, details = result.ErrorDetails });
                }
                
                return Results.Content(result.Data, "application/json");
            }
            catch (AnthropicApiException ex)
            {
                return Results.BadRequest(new { error = ex.Message, details = ex.Details });
            }
        });

        app.MapPost("/api/messages", async (HttpContext context, IAnthropicMessagingService messagingService, IOptions<ApplicationSettings> options) =>
        {
            var requestId = BaseAnthropicService.GenerateRequestId();
            var settings = options.Value;
            
            if (!context.Request.Headers.TryGetValue("x-api-key", out var apiKeyValues) || 
                string.IsNullOrEmpty(apiKeyValues.FirstOrDefault()))
            {
                return Results.BadRequest(new { error = "API key required" });
            }

            var apiKey = apiKeyValues.First()!;
            
            if (apiKey.Length < settings.Features.Validation.MinApiKeyLength)
            {
                return Results.BadRequest(new { error = "Invalid API key format" });
            }
            
            try
            {
                var messageRequest = await context.Request.ReadFromJsonAsync<MessageRequest>();
                if (messageRequest == null)
                {
                    return Results.BadRequest(new { error = "Invalid request body" });
                }

                if (string.IsNullOrEmpty(messageRequest.Model))
                {
                    return Results.BadRequest(new { error = "Model is required" });
                }

                if (messageRequest.Messages == null || !messageRequest.Messages.Any())
                {
                    return Results.BadRequest(new { error = "Messages are required" });
                }

                foreach (var message in messageRequest.Messages)
                {
                    if (message.Content.Length > settings.Features.Validation.MaxMessageLength)
                    {
                        return Results.BadRequest(new { error = $"Message too long (max {settings.Features.Validation.MaxMessageLength} characters)" });
                    }
                }

                if (messageRequest.MaxTokens <= 0)
                {
                    return Results.BadRequest(new { error = "MaxTokens must be greater than 0" });
                }

                var result = await messagingService.SendMessageAsync(apiKey, messageRequest, requestId);
                
                if (!result.Success)
                {
                    return Results.BadRequest(new { error = result.ErrorMessage, details = result.ErrorDetails });
                }
                
                return Results.Ok(result.Response);
            }
            catch (JsonException)
            {
                return Results.BadRequest(new { error = "Invalid JSON format" });
            }
            catch (AnthropicApiException ex)
            {
                return Results.BadRequest(new { error = ex.Message, details = ex.Details });
            }
            catch (Exception ex)
            {
                var logger = context.RequestServices.GetRequiredService<ILogger<Program>>();
                logger.LogError(ex, "Unexpected error in messages endpoint {RequestId}", requestId);
                return Results.StatusCode(500);
            }
        });

        app.MapGet("/healthz", () => Results.NoContent());

        return app;
    }
}
