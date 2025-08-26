using Manto.Web.Configuration;
using Manto.Web.Extensions;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddLogging(builder.Environment);
builder.Services.AddMantoServices(builder.Configuration);

var settings = builder.Configuration.GetSection(ApplicationSettings.SectionName).Get<ApplicationSettings>()!;
builder.WebHost.UseUrls($"http://+:{settings.Server.Port}");

var app = builder.Build();

app.ConfigureMiddleware();

app.MapApiEndpoints();

app.Run();

public partial class Program { }
