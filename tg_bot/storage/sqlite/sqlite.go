package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"tg_game_wishlist/lib/e"
	"tg_game_wishlist/storage"
)

type Storage struct {
	db *sql.DB
}

func (s *Storage) Add(ctx context.Context, w *storage.Wishlist) (err error) {
	defer func() { err = e.WrapIfNil("can't add wishlist", err) }()
	//Получение или создание пользователя
	userId, err := s.getOrCreateUser(ctx, w.User.Name)
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

	q := `INSERT INTO wishlist (game_id, user_id) VALUES (?,?)`

	_, err = s.db.ExecContext(ctx, q, gameId, userId)
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

func (s *Storage) addUser(ctx context.Context, userName string) (int, error) {
	q := `INSERT INTO user (name) VALUES(?)`

	res, err := s.db.ExecContext(ctx, q, userName)
	if err != nil {
		return -1, e.Wrap("can't create user", err)
	}

	userId, err := res.LastInsertId()
	if err != nil {
		return -1, e.Wrap("can't get last user id", err)
	}

	return int(userId), nil
}

func (s *Storage) getOrCreateUser(ctx context.Context, userName string) (int, error) {
	userId, err := s.userId(ctx, userName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return -1, err
	}

	if userId < 0 {
		userId, err = s.addUser(ctx, userName)
		if err != nil {
			return -1, err
		}
	}

	return userId, nil
}

func (s *Storage) gameId(ctx context.Context, g storage.Game) (int, error) {
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

func (s *Storage) addGame(ctx context.Context, g storage.Game) (int, error) {
	q := `INSERT INTO game (name, source, external_url, release_date) VALUES(?,?,?,?)`

	res, err := s.db.ExecContext(ctx, q, g.Name, g.Source, g.ExternalURL, g.ReleaseDate)
	if err != nil {
		return -1, e.Wrap("can't add game", err)
	}

	gameId, err := res.LastInsertId()
	if err != nil {
		return -1, e.Wrap("can't get last game id", err)
	}

	return int(gameId), nil
}

func (s *Storage) getOrCreateGame(ctx context.Context, g storage.Game) (int, error) {
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

func (s *Storage) getGamesFromSqliteQuery(ctx context.Context, query string, args ...any) ([]storage.Game, error) {
	rows, err := s.db.QueryContext(ctx, query, args)
	if err != nil {
		return nil, e.Wrap("can't select games", err)
	}
	defer rows.Close()

	var games []storage.Game

	for rows.Next() {
		var g storage.Game
		var releaseDate sql.NullTime
		var externalURL sql.NullString

		err = rows.Scan(&g.Id, &g.Name, &releaseDate, &g.Source, &externalURL)
		if err != nil {
			return nil, e.Wrap("can't scan game", err)
		}

		if releaseDate.Valid {
			g.ReleaseDate = releaseDate.Time
		}
		if externalURL.Valid {
			g.ExternalURL = externalURL.String
		}

		games = append(games, g)
	}

	if err := rows.Err(); err != nil {
		return nil, e.Wrap("rows iteration error", err)
	}

	return games, nil
}

func (s *Storage) GetAll(ctx context.Context, u *storage.User) ([]storage.Game, error) {
	q := `
		SELECT g.id, g.name, g.release_date, g.source, g.external_url
		FROM wishlist w
		INNER JOIN game g ON w.game_id = g.id
		WHERE w.user_id = ?
		ORDER BY w.added_at DESC
	`

	games, err := s.getGamesFromSqliteQuery(ctx, q, u.Id)
	if err != nil {
		return nil, e.Wrap("can't get all games", err)
	}

	return games, nil
}

func (s *Storage) GetReleased(ctx context.Context, u *storage.User) ([]storage.Game, error) {
	q := `
		SELECT g.id, g.name, g.release_date, g.source, g.external_url
		FROM wishlist w
		INNER JOIN game g ON w.game_id = g.id
		WHERE w.user_id = ? AND g.release_date <= date('now')
		ORDER BY w.added_at DESC
	`

	games, err := s.getGamesFromSqliteQuery(ctx, q, u.Id)
	if err != nil {
		return nil, e.Wrap("can't get released games", err)
	}

	return games, nil
}

func (s *Storage) GetUnreleased(ctx context.Context, u *storage.User) ([]storage.Game, error) {
	q := `
		SELECT g.id, g.name, g.release_date, g.source, g.external_url
		FROM wishlist w
		INNER JOIN game g ON w.game_id = g.id
		WHERE w.user_id = ? AND g.release_date > date('now')
		ORDER BY w.added_at DESC
	`

	games, err := s.getGamesFromSqliteQuery(ctx, q, u.Id)
	if err != nil {
		return nil, e.Wrap("can't get unreleased games", err)
	}

	return games, nil
}

func (s *Storage) Remove(ctx context.Context, w *storage.Wishlist) error {
	q := `DELETE FROM wishlist WHERE id = ?`

	_, err := s.db.ExecContext(ctx, q, w.Id)
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

		var g storage.Game
		var releaseDate sql.NullTime
		var externalURL sql.NullString

		var u storage.User

		err = rows.Scan(&w.Id, &w.AddedAt, notifiedAt, &g.Id, &externalURL, &g.Source, &g.Name, &releaseDate, &u.Id, &u.Name)
		if err != nil {
			return nil, e.Wrap("can't scan wishlist", err)
		}

		if notifiedAt.Valid {
			w.NotifiedAt = notifiedAt.Time
		}

		if releaseDate.Valid {
			g.ReleaseDate = releaseDate.Time
		}
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
	q := `CREATE TABLE user (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL
		);
		
		CREATE TABLE game (
			id INT PRIMARY_KEY AUTO_INCREMENT,
			external_url VARCHAR(500) NULL
			source VARCHAR(255) NOT NULL
			name VARCHAR(255) NOT NULL
			release_date DATETIME NULL
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			    
			UNIQUE KEY unique_external (source, external_url)
		);
		
		CREATE TABLE wishlist (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			game_id INT NOT NULL,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			notified_at DATETIME NULL,
			
			FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
			FOREIGN KEY (game_id) REFERENCES game(id) ON DELETE CASCADE,
			
			UNIQUE KEY (user_id, game_id)
		)`

	_, err := s.db.ExecContext(ctx, q)
	if err != nil {
		return e.Wrap("can't create table", err)
	}

	return nil
}
