using Microsoft.EntityFrameworkCore;
using Variate.Data;
using Variate.Entities;
using Variate.Mapping;
namespace Variate.Endpoints;

public static class CategoriesEndpoints
{
    public static RouteGroupBuilder MapCategoriesEndpoints(this WebApplication app)
    {
        var group = app.MapGroup("categories");
        const string GetCategoryEndpointName = "GetCategory";

        group.MapGet("/", async (VariateContext dbContext) =>
            await dbContext.Categories.Select(category => category.ToDto())
            .AsNoTracking().ToListAsync()
        );

        group.MapGet("/{id}", async (int id, VariateContext dbContext) => 
        {
            Category? category = await dbContext.Categories.FindAsync(id);

            return category is null ? Results.NotFound() : Results.Ok(category);
        })
        .WithName(GetCategoryEndpointName);

        return group;
    }
}
