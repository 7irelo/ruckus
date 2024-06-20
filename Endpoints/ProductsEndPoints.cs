using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;
using Microsoft.EntityFrameworkCore;
using Variate.Data;
using Variate.Dtos;
using Variate.Entities;
using Variate.Mapping;

namespace Variate.Endpoints
{
    public static class ProductsEndpoints
    {
        public static RouteGroupBuilder MapProductsEndpoints(this WebApplication app)
        {
            RouteGroupBuilder group = app.MapGroup("products").WithParameterValidation();

            const string GetProductEndpointName = "GetProduct";

            group.MapGet("/", async (VariateContext dbContext) =>
            {
                var products = await dbContext.Products
                    .Include(product => product.Category)
                    .Select(product => product.ToProductSummaryDto())
                    .AsNoTracking()
                    .ToListAsync();

                return Results.Ok(products);
            });

            group.MapGet("/{id}", async (int id, VariateContext dbContext) =>
            {
                var product = await dbContext.Products.FindAsync(id);

                return product == null ? Results.NotFound() : Results.Ok(product.ToProductDetailsDto());
            }).WithName(GetProductEndpointName);

            group.MapPost("/", async (CreateProductDto newProduct, VariateContext dbContext) =>
            {
                var product = newProduct.ToEntity();
                dbContext.Products.Add(product);
                await dbContext.SaveChangesAsync();

                return Results.CreatedAtRoute(GetProductEndpointName, new { id = product.Id }, product.ToProductDetailsDto());
            });

            group.MapPut("/{id}", async (int id, UpdateProductDto updatedProduct, VariateContext dbContext) =>
            {
                var existingProduct = await dbContext.Products.FindAsync(id);

                if (existingProduct == null)
                    return Results.NotFound();

                dbContext.Entry(existingProduct).CurrentValues.SetValues(updatedProduct.ToEntity(id));
                await dbContext.SaveChangesAsync();

                return Results.NoContent();
            });

            group.MapDelete("/{id}", async (int id, VariateContext dbContext) =>
            {
                await dbContext.Products
                    .Where(product => product.Id == id)
                    .ExecuteDeleteAsync();

                return Results.NoContent();
            });

            return group;
        }
    }
}
