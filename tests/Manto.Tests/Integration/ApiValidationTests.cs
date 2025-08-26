using System.Net.Http.Json;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.Extensions.DependencyInjection;
using FluentAssertions;
using Manto.Web.Configuration;

namespace Manto.Tests.Integration;

[TestClass]
public class ApiValidationTests
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
                        settings.Features.Validation.MaxMessageLength = 100;
                        settings.Features.Validation.MinApiKeyLength = 10;
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
    public async Task MessagesApi_RejectsInvalidRequests()
    {
        var message = new { model = "claude-3-5-haiku", messages = new[] { new { role = "user", content = "hi" } }, max_tokens = 50 };
        var response = await _client.PostAsJsonAsync("/api/messages", message);
        
        response.StatusCode.Should().Be(System.Net.HttpStatusCode.BadRequest);
        var content = await response.Content.ReadAsStringAsync();
        content.Should().Contain("API key required");

        var request = new HttpRequestMessage(HttpMethod.Post, "/api/messages");
        request.Headers.Add("x-api-key", "short");
        request.Content = JsonContent.Create(message);
        response = await _client.SendAsync(request);
        
        response.StatusCode.Should().Be(System.Net.HttpStatusCode.BadRequest);
        content = await response.Content.ReadAsStringAsync();
        content.Should().Contain("Invalid API key format");

        var badMessage = new { model = "", messages = new object[0], max_tokens = 0 };
        request = new HttpRequestMessage(HttpMethod.Post, "/api/messages");
        request.Headers.Add("x-api-key", "sk-ant-test-key-long-enough");
        request.Content = JsonContent.Create(badMessage);
        response = await _client.SendAsync(request);
        
        response.StatusCode.Should().Be(System.Net.HttpStatusCode.BadRequest);
        content = await response.Content.ReadAsStringAsync();
        content.Should().Contain("Model is required");
    }

    [TestMethod]
    public async Task ModelsApi_RequiresApiKey()
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
}