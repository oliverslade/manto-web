using System.Text.Json;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.Extensions.DependencyInjection;
using FluentAssertions;
using Manto.Web.Configuration;

namespace Manto.Tests.Integration;

[TestClass]
public class AppIntegrationTests
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
                            new() { Name = "anthropic", DisplayName = "Anthropic", DefaultModel = "claude-3-5-haiku-latest" },
                            new() { Name = "openai", DisplayName = "OpenAI", DefaultModel = "gpt-4" }
                        };
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
    public async Task App_ShouldServeStaticFiles()
    {
        var response = await _client.GetAsync("/");
        
        response.Should().BeSuccessful();
        response.Content.Headers.ContentType?.MediaType.Should().Be("text/html");
        
        var content = await response.Content.ReadAsStringAsync();
        content.Should().Contain("<title>");
    }

    [TestMethod]
    public async Task Config_ShouldReturnValidJavaScript()
    {
        var response = await _client.GetAsync("/config.js");
        
        response.Should().BeSuccessful();
        response.Content.Headers.ContentType?.MediaType.Should().Be("application/javascript");
        
        var content = await response.Content.ReadAsStringAsync();
        content.Should().StartWith("window.MantoConfig = ");
        content.Should().Contain("providers");
    }

    [TestMethod]
    public async Task Config_ShouldContainProviders()
    {
        var response = await _client.GetAsync("/config.js");
        var content = await response.Content.ReadAsStringAsync();
        
        content.Should().Contain("anthropic");
        content.Should().Contain("openai");
    }

    [TestMethod]
    public async Task Config_ShouldNotExposeSensitiveData()
    {
        var response = await _client.GetAsync("/config.js");
        var content = await response.Content.ReadAsStringAsync();
        
        content.Should().NotContain("api.anthropic.com");
        content.Should().NotContain("8080");
        content.Should().NotContain("EnableHsts");
    }

    [TestMethod]
    public async Task Health_ShouldReturnNoContent()
    {
        var response = await _client.GetAsync("/healthz");
        
        response.Should().BeSuccessful();
        response.StatusCode.Should().Be(System.Net.HttpStatusCode.NoContent);
    }

    [TestMethod]
    public async Task App_ShouldHaveSecurityHeaders()
    {
        var response = await _client.GetAsync("/");
        
        response.Headers.Should().ContainKey("X-Content-Type-Options");
        response.Headers.Should().ContainKey("X-Frame-Options");
        response.Headers.GetValues("Content-Security-Policy").Should().NotBeEmpty();
    }
}
