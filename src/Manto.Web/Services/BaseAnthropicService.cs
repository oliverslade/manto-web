using System.Text.Json;
using Manto.Web.Configuration;

namespace Manto.Web.Services;

public abstract class BaseAnthropicService
{
    protected readonly HttpClient HttpClient;
    protected readonly ILogger Logger;
    protected readonly ProviderConfiguration ProviderConfig;
    protected readonly JsonSerializerOptions JsonOptions;

    protected BaseAnthropicService(
        HttpClient httpClient, 
        ILogger logger, 
        ProviderConfiguration providerConfig)
    {
        HttpClient = httpClient;
        Logger = logger;
        ProviderConfig = providerConfig;
        JsonOptions = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
            DefaultIgnoreCondition = System.Text.Json.Serialization.JsonIgnoreCondition.WhenWritingNull
        };
    }

    protected HttpRequestMessage CreateRequest(HttpMethod method, string endpoint, string apiKey)
    {
        var request = new HttpRequestMessage(method, $"{ProviderConfig.ApiEndpoint}{endpoint}");
        request.Headers.Add("x-api-key", apiKey);
        request.Headers.Add("anthropic-version", ProviderConfig.ApiVersion);
        request.Headers.Add("User-Agent", "Manto/1.0");
        return request;
    }

    protected async Task<T> ExecuteWithErrorHandling<T>(Func<Task<T>> operation, string requestId, string context)
    {
        try
        {
            return await operation();
        }
        catch (TaskCanceledException ex) when (ex.InnerException is TimeoutException)
        {
            Logger.LogError("Anthropic API timeout. Context: {Context}, RequestId: {RequestId}", 
                context, requestId);
            throw new AnthropicApiException("Request timed out", "The request to Anthropic API timed out");
        }
        catch (HttpRequestException ex)
        {
            Logger.LogError(ex, "Network error calling Anthropic API. Context: {Context}, RequestId: {RequestId}", 
                context, requestId);
            throw new AnthropicApiException("Network error", ex.Message);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Unexpected error calling Anthropic API. Context: {Context}, RequestId: {RequestId}", 
                context, requestId);
            throw new AnthropicApiException("Unexpected error", ex.Message);
        }
    }

    public static string GenerateRequestId()
    {
        return Guid.NewGuid().ToString("N")[..8];
    }
}

public class AnthropicApiException : Exception
{
    public string? Details { get; }

    public AnthropicApiException(string message) : base(message)
    {
    }

    public AnthropicApiException(string message, string? details) : base(message)
    {
        Details = details;
    }

    public AnthropicApiException(string message, Exception innerException) : base(message, innerException)
    {
    }
}
