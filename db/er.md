## ER-диаграмма базы данных

```mermaid

erDiagram
    user ||--o{ wishlist : имеет
    game ||--o{ wishlist : включена
    user {
        id INTEGER PK
        name VARCHAR(255)
        chat_id INTEGER
    }
    game {
        id INTEGER PK
        external_url VARCHAR(500)
        source VARCHAR(255)
        name VARCHAR(255)
        created_at DATETIME
    }
    wishlist {
        id INTEGER PK
        user_id INTEGER FK
        game_id INTEGER FK
        notification_date DATETIME
        created_at DATETIME
        notified_at DATETIME
    }
