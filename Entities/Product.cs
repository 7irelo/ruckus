namespace Variate.Entities;

public class Product
{
    public int Id { get; set; }z
    public required string Name { get; set; }
    public int CategoryId { get; set; }
    public Category? Category{ get; set; }
    public decimal Price { get; set; }
    public DateOnly Release { get; set; }
}
