using System.Text;
using System.Text.Json;
using Manto.Web.Configuration;
using Microsoft.Extensions.Options;

namespace Manto.Web.Services;

public class AnthropicMessagingService : BaseAnthropicService, IAnthropicMessagingService
{
    private readonly ProviderConfiguration _anthropicProvider;

    public AnthropicMessagingService(
        HttpClient httpClient, 
        IOptions<ApplicationSettings> settings, 
        ILogger<AnthropicMessagingService> logger)
        : base(httpClient, logger, GetAnthropicProvider(settings.Value))
    {
        _anthropicProvider = GetAnthropicProvider(settings.Value);
    }

    private static ProviderConfiguration GetAnthropicProvider(ApplicationSettings settings)
    {
        var provider = settings.Features.SupportedProviders.FirstOrDefault(p => p.Name == "anthropic");
        if (provider == null)
        {
            throw new InvalidOperationException("Anthropic provider not configured");
        }
        return provider;
    }

    public async Task<MessageResult> SendMessageAsync(string apiKey, MessageRequest request, string requestId)
    {
        return await ExecuteWithErrorHandling(async () =>
        {
            using var httpRequest = CreateRequest(HttpMethod.Post, "/v1/messages", apiKey);
            
            var jsonContent = JsonSerializer.Serialize(request, JsonOptions);
            httpRequest.Content = new StringContent(jsonContent, Encoding.UTF8, "application/json");

            Logger.LogDebug("Sending message to Anthropic API {RequestId}", requestId);

            var response = await HttpClient.SendAsync(httpRequest);

            if (response.IsSuccessStatusCode)
            {
                var responseContent = await response.Content.ReadAsStringAsync();
                var messageResponse = JsonSerializer.Deserialize<MessageResponse>(responseContent, JsonOptions);

                if (messageResponse == null)
                {
                    Logger.LogError("Failed to deserialize message response {RequestId}", requestId);
                    return new MessageResult
                    {
                        Success = false,
                        ErrorMessage = "Invalid response format",
                        ErrorDetails = "Failed to parse message response"
                    };
                }

                Logger.LogDebug("Message sent successfully {RequestId}, tokens: {InputTokens}+{OutputTokens}",
                    requestId, messageResponse.Usage.InputTokens, messageResponse.Usage.OutputTokens);

                return new MessageResult
                {
                    Success = true,
                    Response = messageResponse
                };
            }
            else
            {
                var errorContent = await response.Content.ReadAsStringAsync();
                Logger.LogError("Anthropic API error {RequestId}: {StatusCode} - {Error}", 
                    requestId, response.StatusCode, errorContent);

                string errorMessage = response.StatusCode switch
                {
                    System.Net.HttpStatusCode.Unauthorized => "Invalid API key",
                    System.Net.HttpStatusCode.BadRequest => "Invalid request format",
                    System.Net.HttpStatusCode.TooManyRequests => "Rate limit exceeded",
                    System.Net.HttpStatusCode.InternalServerError => "Service temporarily unavailable",
                    _ => "Failed to send message"
                };

                return new MessageResult
                {
                    Success = false,
                    ErrorMessage = errorMessage,
                    ErrorDetails = errorContent
                };
            }
        }, requestId, "SendMessage");
    }
}
