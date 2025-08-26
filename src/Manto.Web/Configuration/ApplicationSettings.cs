namespace Manto.Web.Configuration;

public class ApplicationSettings
{
    public const string SectionName = "ApplicationSettings";
    
    public ServerSettings Server { get; set; } = new();
    public SecuritySettings Security { get; set; } = new();
    public FeatureSettings Features { get; set; } = new();
}

public class ServerSettings
{
    public int Port { get; set; } = 8080;
    public string AllowedHosts { get; set; } = "*";
}

public class SecuritySettings
{
    public bool EnableHsts { get; set; } = true;
    public List<string> AllowedApiEndpoints { get; set; } = new();
}

public class FeatureSettings
{
    public List<ProviderConfiguration> SupportedProviders { get; set; } = new();
    public ApiSettings Api { get; set; } = new();
    public ValidationSettings Validation { get; set; } = new();
    public ModelSettings Models { get; set; } = new();
}

public class ProviderConfiguration
{
    public string Name { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public string ApiEndpoint { get; set; } = string.Empty;
    public string ApiVersion { get; set; } = string.Empty;
}

public class ApiSettings
{
    public string AnthropicKeyPrefix { get; set; } = "sk-ant-";
    public string PreferredModelId { get; set; } = "claude-3-5-haiku";
    public EndpointSettings Endpoints { get; set; } = new();
}

public class EndpointSettings
{
    public string Models { get; set; } = "/api/models";
    public string Messages { get; set; } = "/api/messages";
}

public class ValidationSettings
{
    public int MaxMessageLength { get; set; } = 4000;
    public int MinApiKeyLength { get; set; } = 10;
}

public class ModelSettings
{
    public int MaxTokens { get; set; } = 1024;
    public double Temperature { get; set; } = 0.7;
    public string SystemMessage { get; set; } = "Please be concise in your responses unless asked otherwise. When explaining concepts, prefer tables and short paragraphs.";
}
