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
    private readonly string _cspHeader;
    private static readonly Dictionary<string, string> _staticHeaders = new()
    {
        ["X-Content-Type-Options"] = "nosniff",
        ["Referrer-Policy"] = "no-referrer",
        ["Permissions-Policy"] = "geolocation=()",
        ["X-Frame-Options"] = "DENY",
        ["Cross-Origin-Opener-Policy"] = "same-origin",
        ["Cross-Origin-Resource-Policy"] = "same-site",
        ["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains; preload",
        ["X-XSS-Protection"] = "1; mode=block",
        ["Cross-Origin-Embedder-Policy"] = "require-corp"
    };

    public SecurityHeadersMiddleware(RequestDelegate next, IOptions<ApplicationSettings> settings)
    {
        _next = next;
        
        var allowedEndpoints = string.Join(" ", settings.Value.Security.AllowedApiEndpoints);
        _cspHeader = "default-src 'self'; " +
                    $"connect-src 'self' {allowedEndpoints}; " +
                    "style-src 'self' 'unsafe-inline'; " +
                    "script-src 'self'; " +
                    "img-src 'self'; " +
                    "object-src 'none'; " +
                    "base-uri 'self'";
    }

    public async Task InvokeAsync(HttpContext context)
    {
        foreach (var header in _staticHeaders)
        {
            context.Response.Headers.Append(header.Key, header.Value);
        }
        
        context.Response.Headers.Append("Content-Security-Policy", _cspHeader);

        await _next(context);
    }
}