using System.ComponentModel.DataAnnotations;

namespace Variate.Dtos;

public record class UpdateProductDto(
    [Required][StringLength(50)] string Name, 
    int CategoryId,
    [Range(0, 10000)] decimal Price, 
    DateOnly Release);