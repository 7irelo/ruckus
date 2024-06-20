#include <iostream>
#include <vector>
#include <algorithm>
#include <functional>
#include <memory>

class Vinyl
{
private:
    int id_;
    std::string artist_;
    std::string name_;
public:
    Vinyl(int id, std::string artist, std::string name) : id_(id), artist_(artist), name_(name) {}
    int getId() const { return id_; }
    std::string getName() const { return name_; }
};

int main(void)
{
    std::vector<std::unique_ptr<Vinyl>> vinyls;
    vinyls.emplace_back(std::make_unique<Vinyl>(1, "Erykah Bady", "Mama's Gun"));
    vinyls.emplace_back(std::make_unique<Vinyl>(4, "Kendrick Lamar", "Good Kid, M.A.AD City"));
    vinyls.emplace_back(std::make_unique<Vinyl>(5, "Frank Ocean", "Blonde."));
    vinyls.emplace_back(std::make_unique<Vinyl>(2, "The Weeknd", "After Hours"));
    vinyls.emplace_back(std::make_unique<Vinyl>(3, "Kenny Rogers", "The Dealer"));
    std::sort(vinyls.begin(), vinyls.end(), [](const std::unique_ptr<Vinyl> &a, const std::unique_ptr<Vinyl> &b)
    {
        return a->getId() < b->getId();
    });
    std::function<void(const Vinyl&)>
    printVynil = [](const Vinyl &vinl) 
    {
        std::cout << vinl.getName() << std::endl;
    };
}