using Microsoft.Extensions.Options;
using Manto.Web.Configuration;

namespace Manto.Web.Extensions;

public static class SecurityHeadersExtensions
{
    public static IApplicationBuilder UseSecurityHeaders(this IApplicationBuilder app)
    {
        return app.UseMiddleware<SecurityHeadersMiddleware>();
    }
}

public class SecurityHeadersMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ApplicationSettings _settings;

    public SecurityHeadersMiddleware(RequestDelegate next, IOptions<ApplicationSettings> settings)
    {
        _next = next;
        _settings = settings.Value;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        context.Response.Headers.Append("X-Content-Type-Options", "nosniff");
        context.Response.Headers.Append("Referrer-Policy", "no-referrer");
        context.Response.Headers.Append("Permissions-Policy", "geolocation=()");
        context.Response.Headers.Append("X-Frame-Options", "DENY");
        context.Response.Headers.Append("Cross-Origin-Opener-Policy", "same-origin");
        context.Response.Headers.Append("Cross-Origin-Resource-Policy", "same-site");

        var allowedEndpoints = string.Join(" ", _settings.Security.AllowedApiEndpoints);
        context.Response.Headers.Append("Content-Security-Policy",
            "default-src 'self'; " +
            $"connect-src 'self' {allowedEndpoints}; " +
            "style-src 'self' 'unsafe-inline'; " +
            "script-src 'self'; " +
            "img-src 'self'; " +
            "object-src 'none'; " +
            "base-uri 'self'");

        context.Response.Headers.Append("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload");
        context.Response.Headers.Append("X-XSS-Protection", "1; mode=block");
        context.Response.Headers.Append("Cross-Origin-Embedder-Policy", "require-corp");

        await _next(context);
    }
}