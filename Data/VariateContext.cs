using Microsoft.EntityFrameworkCore;
using Variate.Entities;

namespace Variate.Data;

public class VariateContext(DbContextOptions<VariateContext> options): DbContext(options)
{
    public DbSet<Product> Products => Set<Product>();

    public DbSet<Category> Categories => Set<Category>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.Entity<Category>().HasData(
            new {Id = 1, Name = "Electronics"},
            new {Id = 2, Name = "Home and kitchen"},
            new {Id = 3, Name = "Fashion and beauty"},
            new {Id = 4, Name = "Toys and games"},
            new {Id = 5, Name = "Books and DVDs"},
            new {Id = 6, Name = "Baby Products"},
            new {Id = 7, Name = "Outdoor and sports equipment"},
            new {Id = 8, Name = "Health and wellness products"},
            new {Id = 9, Name = "Arts and craft supplies"},
            new {Id = 10, Name = "Musical Instruments"}
        );
    }
}
