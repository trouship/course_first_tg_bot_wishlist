package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"tg_game_wishlist/lib/e"
	"tg_game_wishlist/storage"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func (s *Storage) IsExists(ctx context.Context, w *storage.Wishlist) (res bool, err error) {
	defer func() { err = e.WrapIfNil("can't check if exists wishlist", err) }()

	userId, err := s.userId(ctx, w.User.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	gameId, err := s.gameId(ctx, w.Game)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	q := `
		SELECT COUNT(*) 
		FROM wishlist 
		WHERE user_id = ? AND game_id = ?
	`

	var count int

	err = s.db.QueryRowContext(ctx, q, userId, gameId).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return count > 0, nil
}

func (s *Storage) GetUserByName(ctx context.Context, userName string) (*storage.User, error) {
	q := `
		SELECT id, name
		FROM user 
		WHERE name = ?
	`

	var id int
	var name string

	err := s.db.QueryRowContext(ctx, q, userName).Scan(&id, &name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNoUser
		}
		return nil, err
	}

	return &storage.User{
		Id:   id,
		Name: name,
	}, nil
}

func (s *Storage) Add(ctx context.Context, w *storage.Wishlist) (err error) {
	defer func() { err = e.WrapIfNil("can't add wishlist", err) }()
	//Получение или создание пользователя
	userId, err := s.getOrCreateUser(ctx, w.User.Name, w.User.ChatId)
	if err != nil {
		return err
	}
	w.User.Id = userId

	//Получение или создание игры
	gameId, err := s.getOrCreateGame(ctx, w.Game)
	if err != nil {
		return err
	}
	w.Game.Id = gameId

	q := `INSERT INTO wishlist (game_id, user_id, expected_release_date) VALUES (?,?,?)`

	_, err = s.db.ExecContext(ctx, q, gameId, userId, w.ExpectedReleaseDate)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) userId(ctx context.Context, userName string) (int, error) {
	q := `SELECT id FROM user WHERE name = ?`

	var id int

	err := s.db.QueryRowContext(ctx, q, userName).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, err
		}
		return -1, e.Wrap("can't check if user exists", err)
	}

	return id, nil
}

func (s *Storage) addUser(ctx context.Context, userName string, chatId int) (int, error) {
	q := `INSERT INTO user (name, chat_id) VALUES(?, ?)`

	res, err := s.db.ExecContext(ctx, q, userName, chatId)
	if err != nil {
		return -1, e.Wrap("can't create user", err)
	}

	userId, err := res.LastInsertId()
	if err != nil {
		return -1, e.Wrap("can't get last user id", err)
	}

	return int(userId), nil
}

func (s *Storage) getOrCreateUser(ctx context.Context, userName string, chatId int) (int, error) {
	userId, err := s.userId(ctx, userName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return -1, err
	}

	if userId < 0 {
		userId, err = s.addUser(ctx, userName, chatId)
		if err != nil {
			return -1, err
		}
	}

	return userId, nil
}

func (s *Storage) gameId(ctx context.Context, g *storage.Game) (int, error) {
	q := `SELECT id 
		  FROM game 
		  WHERE name = ? AND external_url = ? AND source = ?`

	var id int
	err := s.db.QueryRowContext(ctx, q, g.Name, g.ExternalURL, g.Source).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, err
		}
		return -1, e.Wrap("can't check if game exists", err)
	}

	return id, nil
}

func (s *Storage) addGame(ctx context.Context, g *storage.Game) (int, error) {
	q := `INSERT INTO game (name, source, external_url) VALUES(?,?,?)`

	res, err := s.db.ExecContext(ctx, q, g.Name, g.Source, g.ExternalURL)
	if err != nil {
		return -1, e.Wrap("can't add game", err)
	}

	gameId, err := res.LastInsertId()
	if err != nil {
		return -1, e.Wrap("can't get last game id", err)
	}

	return int(gameId), nil
}

func (s *Storage) getOrCreateGame(ctx context.Context, g *storage.Game) (int, error) {
	gameId, err := s.gameId(ctx, g)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return -1, err
	}

	if gameId < 0 {
		gameId, err = s.addGame(ctx, g)
		if err != nil {
			return -1, err
		}
	}

	return gameId, nil
}

func (s *Storage) getWishlistFromSqliteQuery(ctx context.Context, query string, args ...any) ([]storage.Wishlist, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, e.Wrap("can't select wishlist", err)
	}
	defer rows.Close()

	var wishlist []storage.Wishlist

	for rows.Next() {
		var w storage.Wishlist
		var expectedReleaseDate sql.NullTime
		var createdDate sql.NullTime
		var notifiedDate sql.NullTime

		var g storage.Game
		var externalURL sql.NullString

		var u storage.User

		err = rows.Scan(&w.Id, &expectedReleaseDate, &notifiedDate, &createdDate, &g.Id, &g.Name, &g.Source, &externalURL, &u.Id, &u.Name)
		if err != nil {
			return nil, e.Wrap("can't scan game", err)
		}

		if createdDate.Valid {
			w.AddedAt = createdDate.Time
		}
		if notifiedDate.Valid {
			w.NotifiedAt = notifiedDate.Time
		}
		if expectedReleaseDate.Valid {
			w.ExpectedReleaseDate = expectedReleaseDate.Time
		}
		if externalURL.Valid {
			g.ExternalURL = externalURL.String
		}

		w.Game = &g
		w.User = &u

		wishlist = append(wishlist, w)
	}

	if err := rows.Err(); err != nil {
		return nil, e.Wrap("rows iteration error", err)
	}

	return wishlist, nil
}

func (s *Storage) GetAll(ctx context.Context, u *storage.User) ([]storage.Wishlist, error) {
	q := `
		SELECT w.id, w.expected_release_date, w.notified_at, w.created_at, g.id, g.name, g.source, g.external_url, u.id, u.name
		FROM wishlist w
		INNER JOIN game g ON w.game_id = g.id
		INNER JOIN user u on w.user_id = u.id
		WHERE w.user_id = ?
		ORDER BY g.name ASC
	`

	wishlist, err := s.getWishlistFromSqliteQuery(ctx, q, u.Id)
	if err != nil {
		return nil, e.Wrap("can't get all games", err)
	}

	return wishlist, nil
}

func (s *Storage) GetReleased(ctx context.Context, u *storage.User) ([]storage.Wishlist, error) {
	q := `
		SELECT w.id, w.expected_release_date, w.notified_at, w.created_at, g.id, g.name, g.source, g.external_url, u.id, u.name
		FROM wishlist w
		INNER JOIN game g ON w.game_id = g.id
		INNER JOIN user u on w.user_id = u.id
		WHERE w.user_id = ? AND g.release_date <= date('now')
		ORDER BY g.name ASC
	`

	wishlist, err := s.getWishlistFromSqliteQuery(ctx, q, u.Id)
	if err != nil {
		return nil, e.Wrap("can't get released games", err)
	}

	return wishlist, nil
}

func (s *Storage) GetUnreleased(ctx context.Context, u *storage.User) ([]storage.Wishlist, error) {
	q := `
		SELECT w.id, w.expected_release_date, w.notified_at, w.created_at, g.id, g.name, g.source, g.external_url, u.id, u.name
		FROM wishlist w
		INNER JOIN game g ON w.game_id = g.id
		INNER JOIN user u on w.user_id = u.id
		WHERE w.user_id = ? AND g.release_date > date('now')
		ORDER BY g.name ASC
	`

	wishlist, err := s.getWishlistFromSqliteQuery(ctx, q, u.Id)
	if err != nil {
		return nil, e.Wrap("can't get unreleased games", err)
	}

	return wishlist, nil
}

func (s *Storage) Remove(ctx context.Context, wishlistId int) error {
	q := `DELETE FROM wishlist WHERE id = ?`

	_, err := s.db.ExecContext(ctx, q, wishlistId)
	if err != nil {
		return e.Wrap("can't remove wishlist", err)
	}

	return nil
}

func (s *Storage) GetToNotify(ctx context.Context) ([]storage.Wishlist, error) {
	q := `
		SELECT w.id, w.added_at, w.notified_at, g.id, g.external_url, g.source, g.name, g.release_date, u.id, u.name
    	FROM wishlist w
		INNER JOIN game g ON w.game_id = g.id
    	INNER JOIN user u ON w.user_id = u.id
		WHERE w.notified_at NULL AND g.release_date >= date('now')
    `

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, e.Wrap("can't get wishlist to notify", err)
	}
	defer rows.Close()

	var wishlist []storage.Wishlist

	for rows.Next() {
		var w storage.Wishlist
		var notifiedAt sql.NullTime

		var g *storage.Game
		var releaseDate sql.NullTime
		var externalURL sql.NullString

		var u *storage.User

		err = rows.Scan(&w.Id, &w.AddedAt, notifiedAt, &g.Id, &externalURL, &g.Source, &g.Name, &releaseDate, &u.Id, &u.Name)
		if err != nil {
			return nil, e.Wrap("can't scan wishlist", err)
		}

		if notifiedAt.Valid {
			w.NotifiedAt = notifiedAt.Time
		}

		//if releaseDate.Valid {
		//	g.ReleaseDate = releaseDate.Time
		//}
		if externalURL.Valid {
			g.ExternalURL = externalURL.String
		}

		w.Game = g
		w.User = u

		wishlist = append(wishlist, w)
	}

	if err := rows.Err(); err != nil {
		return nil, e.Wrap("rows iteration error", err)
	}

	return wishlist, nil
}

func (s *Storage) Notify(ctx context.Context, w *storage.Wishlist) error {
	q := `UPDATE wishlist SET notified_at = date('now') WHERE id = ?`

	_, err := s.db.ExecContext(ctx, q, w.Id)
	if err != nil {
		return e.Wrap("can't edit notify wishlist", err)
	}

	return nil
}

func New(path string) (*Storage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, e.Wrap("can't open database", err)
	}

	if err := db.Ping(); err != nil {
		return nil, e.Wrap("can't connect to database", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Init(ctx context.Context) error {
	//TODO Add expected date in wishlist + preferred platform (nullable) if there is a platform on which it exists and there is a platform where it does not
	q := `
		CREATE TABLE IF NOT EXISTS user (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(255) NOT NULL,
		    chat_id INTEGER NOT NULL,
		    
		    UNIQUE(name, chat_id)
		);
		
		CREATE TABLE IF NOT EXISTS game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			external_url VARCHAR(500) NULL,
			source VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			
			UNIQUE(source, external_url, name)
		);
		
		CREATE TABLE IF NOT EXISTS wishlist (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			game_id INTEGER NOT NULL,
			expected_release_date DATETIME NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			notified_at DATETIME NULL,
			
			FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
			FOREIGN KEY (game_id) REFERENCES game(id) ON DELETE CASCADE,
			
			UNIQUE(user_id, game_id)
		);
	`

	_, err := s.db.ExecContext(ctx, q)
	if err != nil {
		return e.Wrap("can't create table", err)
	}

	return nil
}
