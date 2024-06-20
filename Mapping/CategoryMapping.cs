using Variate.Dto;
using Variate.Dtos;
using Variate.Entities;
namespace Variate.Mapping;


public static class CategoryMapping
{
    public static CategoryDto ToDto(this Category category)
    {
        return new CategoryDto(category.Id, category.Name);
    }
}
