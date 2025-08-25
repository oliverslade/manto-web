var builder = WebApplication.CreateBuilder(args);

var app = builder.Build();

// Security headers middleware
app.Use(async (context, next) =>
{
    context.Response.Headers.Append("X-Content-Type-Options", "nosniff");
    context.Response.Headers.Append("Referrer-Policy", "no-referrer");
    context.Response.Headers.Append("Permissions-Policy", "geolocation=()");
    context.Response.Headers.Append("X-Frame-Options", "DENY");
    context.Response.Headers.Append("Cross-Origin-Opener-Policy", "same-origin");
    context.Response.Headers.Append("Cross-Origin-Resource-Policy", "same-site");
    context.Response.Headers.Append("Content-Security-Policy", "default-src 'self'; connect-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self';");
    
    await next();
});

// Configure pipeline
if (!app.Environment.IsDevelopment())
{
    app.UseHsts();
}

app.UseHttpsRedirection();
app.UseDefaultFiles();
app.UseStaticFiles();

// Health check endpoint
app.MapGet("/healthz", () => Results.NoContent());

app.Run("http://+:8080");
