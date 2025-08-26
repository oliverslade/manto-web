using Manto.Web.Configuration;
using Manto.Web.Services;

namespace Manto.Web.Extensions;

public static class ServiceCollectionExtensions
{
    public static IServiceCollection AddMantoServices(this IServiceCollection services, IConfiguration configuration)
    {
        services.Configure<ApplicationSettings>(
            configuration.GetSection(ApplicationSettings.SectionName));

        services.AddOutputCache();

        services.AddAnthropicServices();

        return services;
    }

    private static IServiceCollection AddAnthropicServices(this IServiceCollection services)
    {
        services.AddHttpClient<IAnthropicApiService, AnthropicApiService>(client =>
        {
            client.Timeout = TimeSpan.FromSeconds(30);
        });

        services.AddHttpClient<IAnthropicMessagingService, AnthropicMessagingService>(client =>
        {
            client.Timeout = TimeSpan.FromSeconds(60);
        });

        services.AddScoped<IAnthropicApiService>(provider =>
        {
            var httpClient = provider.GetRequiredService<HttpClient>();
            var logger = provider.GetRequiredService<ILogger<AnthropicApiService>>();
            var settings = provider.GetRequiredService<Microsoft.Extensions.Options.IOptions<ApplicationSettings>>().Value;
            
            var anthropicProvider = settings.Features.SupportedProviders.FirstOrDefault(p => p.Name == "anthropic");
            if (anthropicProvider == null)
            {
                throw new InvalidOperationException("Anthropic provider not configured");
            }
            
            return new AnthropicApiService(httpClient, logger, anthropicProvider);
        });

        return services;
    }

    public static IServiceCollection AddLogging(this IServiceCollection services, IWebHostEnvironment environment)
    {
        services.AddLogging(builder =>
        {
            builder.ClearProviders();
            builder.AddConsole();
            
            if (environment.IsDevelopment())
            {
                builder.SetMinimumLevel(LogLevel.Information);
            }
            else
            {
                builder.SetMinimumLevel(LogLevel.Warning);
            }
        });

        return services;
    }
}
