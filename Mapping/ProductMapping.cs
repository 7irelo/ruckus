using Variate.Entities;
using Variate.Dtos;
namespace Variate.Mapping;

public static class ProductMapping
{
    public static Product ToEntity(this CreateProductDto product)
    {
        return new Product()
        {
            Name = product.Name,
            CategoryId = product.CategoryId,
            Price = product.Price,
            Release = product.Release
        };
    }


    public static Product ToEntity(this UpdateProductDto product, int id)
    {
        return new Product()
        {
            Id = id,
            Name = product.Name,
            CategoryId = product.CategoryId,
            Price = product.Price,
            Release = product.Release
        };
    }

    public static ProductSummaryDto ToProductSummaryDto(this Product product)
    {
        return new(
            product.Id,
            product.Name,
            product.Category!.Name,
            product.Price,
            product.Release
        );
    }

    public static ProductDetailsDto ToProductDetailsDto(this Product product)
    {
        return new(
            product.Id,
            product.Name,
            product.CategoryId,
            product.Price,
            product.Release
        );
    }
}
