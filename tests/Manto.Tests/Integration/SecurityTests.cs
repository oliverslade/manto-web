using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.Extensions.DependencyInjection;
using FluentAssertions;
using Manto.Web.Configuration;

namespace Manto.Tests.Integration;

[TestClass]
public class SecurityTests
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
                        settings.Security.AllowedApiEndpoints = new List<string> { "https://api.anthropic.com" };
                        settings.Features.SupportedProviders = new List<ProviderConfiguration>
                        {
                            new() { Name = "anthropic", DisplayName = "Anthropic", ApiEndpoint = "https://api.anthropic.com", ApiVersion = "2023-06-01" }
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
    public async Task AllRequests_HaveSecurityHeaders()
    {
        var response = await _client.GetAsync("/");

        response.Headers.Should().ContainKey("X-Content-Type-Options");
        response.Headers.Should().ContainKey("X-Frame-Options");
        response.Headers.Should().ContainKey("Content-Security-Policy");
        response.Headers.Should().ContainKey("Strict-Transport-Security");
        
        var csp = string.Join(" ", response.Headers.GetValues("Content-Security-Policy"));
        csp.Should().Contain("https://api.anthropic.com");
        csp.Should().Contain("default-src 'self'");
    }
}