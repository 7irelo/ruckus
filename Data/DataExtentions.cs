using Microsoft.EntityFrameworkCore;

namespace Variate.Data;

public static class DataExtentions
{
    public static async Task MigrateDbAsync(this WebApplication app)
    {
        using var scope = app.Services.CreateScope();
        VariateContext dbContext = scope.ServiceProvider.GetRequiredService<VariateContext>();
        await dbContext.Database.MigrateAsync();
    }
}
