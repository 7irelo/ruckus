using Variate.Endpoints;
using Variate.Data;

WebApplicationBuilder builder = WebApplication.CreateBuilder(args);

string? connString = builder.Configuration.GetConnectionString("Variate");
builder.Services.AddSqlite<VariateContext>(connString);

WebApplication app = builder.Build();

app.MapProductsEndpoints();
app.MapCategoriesEndpoints();

app.MapGet("/", () => "Hello World");

await app.MigrateDbAsync();

app.Run();