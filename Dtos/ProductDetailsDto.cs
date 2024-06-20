namespace Variate.Dtos;

public record class ProductDetailsDto(int Id, string Name, int CategoryId, decimal Price, DateOnly Release);