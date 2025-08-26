using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.Extensions.DependencyInjection;
using FluentAssertions;
using Manto.Web.Configuration;

namespace Manto.Tests.Integration;

[TestClass]
public class ConfigEndpointTests
{
    private WebApplicationFactory<Program> _factory = null!;
    private HttpClient _client = null!;

    [TestInitialize]
    public void Setup()
    {
        _factory = new WebApplicationFactory<Program>()
            .WithWebHostBuilder(builder =>
            {
                builder.ConfigureServices(services =>
                {
                    services.Configure<ApplicationSettings>(settings =>
                    {
                        settings.Features.SupportedProviders = new List<ProviderConfiguration>
                        {
                            new() { Name = "anthropic", DisplayName = "Anthropic", ApiEndpoint = "https://api.anthropic.com", ApiVersion = "2023-06-01" }
                        };
                        settings.Features.Api.AnthropicKeyPrefix = "sk-ant-";
                        settings.Features.Validation.MaxMessageLength = 4000;
                    });
                });
            });

        _client = _factory.CreateClient();
    }

    [TestCleanup]
    public void Cleanup()
    {
        _client?.Dispose();
        _factory?.Dispose();
    }

    [TestMethod]
    public async Task ConfigJs_ReturnsValidConfiguration()
    {
        var response = await _client.GetAsync("/config.js");

        response.Should().BeSuccessful();
        response.Content.Headers.ContentType?.MediaType.Should().Be("application/javascript");
        
        var content = await response.Content.ReadAsStringAsync();
        content.Should().StartWith("window.MantoConfig = ");
        content.Should().Contain("anthropic");
        content.Should().Contain("sk-ant-");
        
        content.Should().NotContain("EnableHsts");
        content.Should().NotContain("AllowedHosts");
        content.Should().NotContain("Port");
        content.Should().NotContain("8080");
    }
}