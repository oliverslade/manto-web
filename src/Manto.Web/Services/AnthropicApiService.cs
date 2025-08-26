using Manto.Web.Configuration;

namespace Manto.Web.Services;

public interface IAnthropicApiService
{
    Task<ApiResult> GetModelsAsync(string apiKey, string requestId);
}

public class AnthropicApiService : BaseAnthropicService, IAnthropicApiService
{
    public AnthropicApiService(HttpClient httpClient, ILogger<AnthropicApiService> logger, ProviderConfiguration providerConfig)
        : base(httpClient, logger, providerConfig)
    {
    }

    public async Task<ApiResult> GetModelsAsync(string apiKey, string requestId)
    {
        return await ExecuteWithErrorHandling(async () =>
        {
            using var request = CreateRequest(HttpMethod.Get, "/v1/models", apiKey);
            var response = await HttpClient.SendAsync(request);

            if (!response.IsSuccessStatusCode)
            {
                var errorContent = await response.Content.ReadAsStringAsync();
                Logger.LogError("Anthropic API error. RequestId: {RequestId}, Status: {StatusCode}, Error: {Error}", 
                    requestId, (int)response.StatusCode, errorContent);
                return ApiResult.Failure("Failed to fetch models", errorContent);
            }

            var modelsContent = await response.Content.ReadAsStringAsync();
            return ApiResult.Success(modelsContent);
        }, requestId, "GetModels");
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
