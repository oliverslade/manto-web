using System.Text.Json;
using Microsoft.Extensions.Options;
using Manto.Web.Configuration;
using Manto.Web.Extensions;
using Manto.Web.Services;

var builder = WebApplication.CreateBuilder(args);

builder.Logging.ClearProviders();
builder.Logging.AddConsole();
if (builder.Environment.IsDevelopment())
{
    builder.Logging.SetMinimumLevel(LogLevel.Information);
}
else
{
    builder.Logging.SetMinimumLevel(LogLevel.Warning);
}

builder.Services.Configure<ApplicationSettings>(
    builder.Configuration.GetSection(ApplicationSettings.SectionName));

builder.Services.AddOutputCache();

builder.Services.AddHttpClient<IAnthropicApiService, AnthropicApiService>(client =>
{
    client.Timeout = TimeSpan.FromSeconds(30);
});

builder.Services.AddScoped<IAnthropicApiService>(provider =>
{
    var httpClient = provider.GetRequiredService<HttpClient>();
    var logger = provider.GetRequiredService<ILogger<AnthropicApiService>>();
    var settings = provider.GetRequiredService<IOptions<ApplicationSettings>>().Value;
    var anthropicProvider = settings.Features.SupportedProviders.FirstOrDefault(p => p.Name == "anthropic");
    
    if (anthropicProvider == null)
    {
        throw new InvalidOperationException("Anthropic provider not configured");
    }
    
    return new AnthropicApiService(httpClient, logger, anthropicProvider);
});

var tempSettings = new ApplicationSettings();
builder.Configuration.GetSection(ApplicationSettings.SectionName).Bind(tempSettings);
builder.WebHost.UseUrls($"http://+:{tempSettings.Server.Port}");

var app = builder.Build();

var logger = app.Services.GetRequiredService<ILogger<Program>>();

logger.LogInformation("Manto starting on port {Port} ({Environment})", 
    tempSettings.Server.Port, app.Environment.EnvironmentName);

app.UseSecurityHeaders();

if (!app.Environment.IsDevelopment())
{
    var appSettings = app.Services.GetRequiredService<IOptions<ApplicationSettings>>().Value;
    if (appSettings.Security.EnableHsts)
    {
        app.UseHsts();
    }
}

app.UseHttpsRedirection();
app.UseOutputCache();
app.UseDefaultFiles();
app.UseStaticFiles();

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
    var requestId = AnthropicApiService.GenerateRequestId();
    
    if (!context.Request.Headers.TryGetValue("x-api-key", out var apiKeyValues) || 
        string.IsNullOrEmpty(apiKeyValues.FirstOrDefault()))
    {
        return Results.BadRequest(new { error = "API key required" });
    }

    var apiKey = apiKeyValues.First()!;
    
    var result = await anthropicService.GetModelsAsync(apiKey, requestId);
    
    if (!result.IsSuccess)
    {
        return Results.BadRequest(new { error = result.ErrorMessage, details = result.ErrorDetails });
    }
    
    return Results.Content(result.Data, "application/json");
});

app.MapGet("/healthz", () => Results.NoContent());

app.Run();

public partial class Program { }
