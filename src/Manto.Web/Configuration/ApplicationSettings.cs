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
}

public class ProviderConfiguration
{
    public string Name { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public string ApiEndpoint { get; set; } = string.Empty;
    public string ApiVersion { get; set; } = string.Empty;
}
