using System.Net.Http.Json;
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
                            new() 
                            { 
                                Name = "anthropic", 
                                DisplayName = "Anthropic", 
                                ApiEndpoint = "https://api.anthropic.com",
                                ApiVersion = "2023-06-01"
                            }
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
    public async Task Config_ReturnsValidConfigurationForFrontend()
    {
        var response = await _client.GetAsync("/config.js");
        
        response.Should().BeSuccessful();
        response.Content.Headers.ContentType?.MediaType.Should().Be("application/javascript");
        
        var content = await response.Content.ReadAsStringAsync();
        content.Should().StartWith("window.MantoConfig = ");
        content.Should().Contain("anthropic");
        
        content.Should().NotContain("8080");
        content.Should().NotContain("EnableHsts");
    }

    [TestMethod]
    public async Task ModelsApi_WithoutApiKey_Returns400()
    {
        var response = await _client.GetAsync("/api/models");
        
        response.StatusCode.Should().Be(System.Net.HttpStatusCode.BadRequest);
        var content = await response.Content.ReadAsStringAsync();
        content.Should().Contain("API key required");
    }

    [TestMethod]
    public async Task ModelsApi_WithInvalidApiKey_Returns400()
    {
        var request = new HttpRequestMessage(HttpMethod.Get, "/api/models");
        request.Headers.Add("x-api-key", "invalid-key");
        
        var response = await _client.SendAsync(request);
        
        response.StatusCode.Should().Be(System.Net.HttpStatusCode.BadRequest);
        var content = await response.Content.ReadAsStringAsync();
        content.Should().Contain("Failed to fetch models");
    }

    [TestMethod]
    public async Task Health_ShouldReturnNoContent()
    {
        var response = await _client.GetAsync("/healthz");
        
        response.Should().BeSuccessful();
        response.StatusCode.Should().Be(System.Net.HttpStatusCode.NoContent);
    }

    [TestMethod]
    public async Task App_StartsWithoutErrors()
    {
        var response = await _client.GetAsync("/healthz");
        response.Should().BeSuccessful();
        
        response = await _client.GetAsync("/");
        response.Should().BeSuccessful();
        
        response = await _client.GetAsync("/config.js");
        response.Should().BeSuccessful();
    }
}
