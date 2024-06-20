namespace Variate.Dtos;

public record class ProductSummaryDto(int Id, string Name, string Category, decimal Price, DateOnly Release);