using System.Text.Json;
using Microsoft.Extensions.Options;
using Manto.Web.Configuration;
using Manto.Web.Extensions;

var builder = WebApplication.CreateBuilder(args);

builder.Services.Configure<ApplicationSettings>(
    builder.Configuration.GetSection(ApplicationSettings.SectionName));

builder.Services.AddOutputCache();

var tempSettings = new ApplicationSettings();
builder.Configuration.GetSection(ApplicationSettings.SectionName).Bind(tempSettings);
builder.WebHost.UseUrls($"http://+:{tempSettings.Server.Port}");

var app = builder.Build();

app.UseSecurityHeaders();

if (!app.Environment.IsDevelopment())
{
    var appSettings = app.Services.GetRequiredService<IOptions<ApplicationSettings>>().Value;
    if (appSettings.Security.EnableHsts)
    {
        app.UseHsts();
    }
}

app.UseHttpsRedirection();
app.UseOutputCache();
app.UseDefaultFiles();
app.UseStaticFiles();

app.MapGet("/config.js", (IOptions<ApplicationSettings> options) =>
{
    var settings = options.Value;
    
    var config = new
    {
        providers = settings.Features.SupportedProviders.Select(p => new
        {
            name = p.Name,
            displayName = p.DisplayName,
            defaultModel = p.DefaultModel
        }).ToArray(),
        version = "1.0.0"
    };

    var jsonConfig = JsonSerializer.Serialize(config, new JsonSerializerOptions
    {
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        WriteIndented = false
    });

    var configScript = $"window.MantoConfig = {jsonConfig};";

    return Results.Content(configScript, "application/javascript");
}).CacheOutput(policy => policy.Expire(TimeSpan.FromMinutes(5)));

app.MapGet("/healthz", () => Results.NoContent());

app.Run();

public partial class Program { }
