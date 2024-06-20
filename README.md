# E-Commerce Web App API

## Overview

Welcome to the **E-Commerce Web App API** project. This API provides the backend functionality for an e-commerce application, including user authentication, product management, order processing, and more. It is built using ASP.NET and SQL Server.

## Features

- **User Authentication and Authorization**
  - Sign Up, Login, and Logout
  - Password Recovery
  - Role-based Access Control

- **Product Management**
  - Create, Read, Update, and Delete (CRUD) Products
  - Product Categories
  - Inventory Management

- **Order Management**
  - Create and Manage Orders
  - Order Status Tracking
  - Payment Processing

- **User Management**
  - User Profiles
  - Address Book Management
  - Order History

- **Search and Filter**
  - Product Search
  - Category and Price Filters

- **Notifications**
  - Email Notifications for Order Confirmations and Updates

## Prerequisites

- **.NET 5.0+**
- **SQL Server 2017+**
- **Visual Studio 2019+**

## Getting Started

### Setting Up the Development Environment

1. **Clone the Repository**:
    ```bash
    git clone https://github.com/7irelo/variate-asp-api.git
    cd variate-asp-api
    ```

2. **Open the Solution**:
    - Open `EcommerceApi.sln` in Visual Studio.

3. **Configure SQL Server**:
    - Create a new SQL Server database named `database_name`.
    - Update the connection string in `appsettings.json` with your SQL Server credentials:
    ```json
    "ConnectionStrings": {
        "DefaultConnection": "Server=your_server_name;Database=database_name;User Id=your_username;Password=your_password;"
    }
    ```

4. **Apply Migrations**:
    - Open the Package Manager Console in Visual Studio.
    - Run the following commands:
    ```bash
    Update-Database
    ```

5. **Run the Application**:
    - Press `F5` or click on the `Run` button in Visual Studio.

### API Endpoints

The following are the main API endpoints available:

#### User Authentication
- **POST /api/auth/register**: Register a new user
- **POST /api/auth/login**: User login
- **POST /api/auth/logout**: User logout
- **POST /api/auth/forgot-password**: Password recovery

#### Product Management
- **GET /api/products**: Get all products
- **GET /api/products/{id}**: Get a specific product by ID
- **POST /api/products**: Create a new product
- **PUT /api/products/{id}**: Update an existing product
- **DELETE /api/products/{id}**: Delete a product

#### Order Management
- **GET /api/orders**: Get all orders
- **GET /api/orders/{id}**: Get a specific order by ID
- **POST /api/orders**: Create a new order
- **PUT /api/orders/{id}**: Update an order
- **DELETE /api/orders/{id}**: Cancel an order

#### User Management
- **GET /api/users/{id}**: Get user profile
- **PUT /api/users/{id}**: Update user profile
- **GET /api/users/{id}/orders**: Get user's order history

### Testing the API

Use tools like **Postman** or **curl** to test the API endpoints. For example, to register a new user:

```bash
POST /api/auth/register
Content-Type: application/json

{
  "username": "johndoe",
  "email": "johndoe@example.com",
  "password": "Password123!"
}
```

## Contributing

We welcome contributions from the community. Please follow these steps to contribute:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature-branch`).
3. Make your changes.
4. Commit your changes (`git commit -m 'Add some feature'`).
5. Push to the branch (`git push origin feature-branch`).
6. Open a pull request.

Ensure your code adheres to the project's coding standards and includes appropriate tests.

## Reporting Issues

If you encounter any issues, please report them on our [Issue Tracker](https://github.com/7irelo/asp.net-e-commerce-api/issues).

## License

This project is licensed under the [MIT License](LICENSE).

## Contact

For any queries or support, please contact us at tirelo.eric@gmail.com.

---

Thank you for using the E-Commerce Web App API. We hope it helps you build a robust and scalable e-commerce application!
