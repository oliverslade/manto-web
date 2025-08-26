using System.Text.Json;
using Manto.Web.Configuration;

namespace Manto.Web.Services;

public interface IAnthropicApiService
{
    Task<ApiResult> GetModelsAsync(string apiKey, string requestId);
}

public class AnthropicApiService : IAnthropicApiService
{
    private readonly HttpClient _httpClient;
    private readonly ILogger<AnthropicApiService> _logger;
    private readonly ProviderConfiguration _providerConfig;

    public AnthropicApiService(HttpClient httpClient, ILogger<AnthropicApiService> logger, ProviderConfiguration providerConfig)
    {
        _httpClient = httpClient;
        _logger = logger;
        _providerConfig = providerConfig;
    }

    public async Task<ApiResult> GetModelsAsync(string apiKey, string requestId)
    {
        try
        {
            using var request = new HttpRequestMessage(HttpMethod.Get, $"{_providerConfig.ApiEndpoint}/v1/models");
            request.Headers.Add("x-api-key", apiKey);
            request.Headers.Add("anthropic-version", _providerConfig.ApiVersion);

            var stopwatch = System.Diagnostics.Stopwatch.StartNew();
            var response = await _httpClient.SendAsync(request);
            stopwatch.Stop();

            if (stopwatch.ElapsedMilliseconds > 2000)
            {
                _logger.LogWarning("Slow Anthropic API response. RequestId: {RequestId}, Duration: {Duration}ms", 
                    requestId, stopwatch.ElapsedMilliseconds);
            }

            if (!response.IsSuccessStatusCode)
            {
                var errorContent = await response.Content.ReadAsStringAsync();
                _logger.LogError("Anthropic API error. RequestId: {RequestId}, Status: {StatusCode}, Error: {Error}", 
                    requestId, (int)response.StatusCode, errorContent);
                return ApiResult.Failure($"Failed to fetch models", errorContent);
            }

            var modelsContent = await response.Content.ReadAsStringAsync();
            return ApiResult.Success(modelsContent);
        }
        catch (TaskCanceledException)
        {
            _logger.LogError("Anthropic API timeout. RequestId: {RequestId}", requestId);
            return ApiResult.Failure("Request timed out");
        }
        catch (HttpRequestException ex)
        {
            _logger.LogError(ex, "Network error calling Anthropic API. RequestId: {RequestId}", requestId);
            return ApiResult.Failure("Network error", ex.Message);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Unexpected error fetching models. RequestId: {RequestId}", requestId);
            return ApiResult.Failure("Failed to fetch models", ex.Message);
        }
    }

    public static string GenerateRequestId()
    {
        return Guid.NewGuid().ToString("N")[..8];
    }
}

public class ApiResult
{
    public bool IsSuccess { get; private set; }
    public string Data { get; private set; } = string.Empty;
    public string ErrorMessage { get; private set; } = string.Empty;
    public string? ErrorDetails { get; private set; }

    private ApiResult() { }

    public static ApiResult Success(string data)
    {
        return new ApiResult { IsSuccess = true, Data = data };
    }

    public static ApiResult Failure(string errorMessage, string? errorDetails = null)
    {
        return new ApiResult { IsSuccess = false, ErrorMessage = errorMessage, ErrorDetails = errorDetails };
    }
}
