package repositories

import (
	"context"
	"log"
	"strconv"

	"github.com/alanqchen/Bear-Post/backend/database"
	"github.com/alanqchen/Bear-Post/backend/models"
	"github.com/jackc/pgx/v4"
)

// PostRepository interface
type PostRepository interface {
	Create(p *models.Post) error
	GetAll() ([]*models.Post, error)
	FindByID(id int) (*models.Post, error)
	FindByIDAdmin(id int) (*models.Post, error)
	FindBySlug(slug string) (*models.Post, error)
	FindBySlugAdmin(slug string) (*models.Post, error)
	Exists(slug string) bool
	Delete(id int) error
	Update(p *models.Post) error
	Paginate(maxID int, perPage int, tags []string) ([]*models.Post, int, error)
	PaginateAdmin(maxID int, perPage int, tags []string) ([]*models.Post, int, error)
	GetTotalPostCount() (int, error)
	GetPublicPostCount() (int, error)
	ResetSeq() error
	GetLastID() (int, error)
	GetLastIDAdmin() (int, error)
	SearchQuery(string, []string) ([]*models.Post, error)
}

type postRepository struct {
	*database.Postgres
}

// NewPostRepository - creates a post repository instance
func NewPostRepository(db *database.Postgres) PostRepository {
	return &postRepository{db}
}

// Create creates a new post in the database
func (pr *postRepository) Create(p *models.Post) error {
	exists := pr.Exists(p.Slug)
	if exists {
		err := pr.createWithSlugCount(p)
		if err != nil {
			return err
		}

		return nil
	}
	/*
		_, err := pr.Conn.Prepare(context.Background(), "post-query", "INSERT INTO post_schema.post VALUES (default, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id")
		if err != nil {
			log.Println(err)
			return err
		}
	*/

	var pID int

	err := pr.Pool.QueryRow(
		context.Background(),
		"INSERT INTO post_schema.post VALUES (default, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id",
		p.Title, p.Slug, p.Body, p.CreatedAt.UTC(), nil, p.Tags, p.Hidden, p.AuthorID, p.FeatureImgURL, p.Subtitle, p.Views,
	).Scan(&pID)

	if err != nil {
		log.Println(err)
		return err
	}

	p.ID = pID

	return nil
}

// Delete deletes the post with the given ID in the database
func (pr *postRepository) Delete(id int) error {
	row, err := pr.Pool.Query(context.Background(), "DELETE FROM post_schema.post WHERE id=$1", id)
	defer row.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// Exists checks if a post with the slug already exists in the database
func (pr *postRepository) Exists(slug string) bool {
	var exists bool
	err := pr.Pool.QueryRow(context.Background(), "SELECT EXISTS (SELECT id FROM post_schema.post WHERE slug=$1)", slug).Scan(&exists)
	if err != nil {
		log.Printf("[POST REPO]: Exists err %v", err)
		return true
	}

	return exists
}

// This is a private function to be used in cases where a slug already exists
func (pr *postRepository) createWithSlugCount(p *models.Post) error {

	var count int
	err := pr.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM post_schema.post WHERE slug LIKE $1", p.Slug+"%").Scan(&count)
	if err != nil {
		log.Println(err)
		return err
	}
	counter := strconv.Itoa(count + 1)
	/*
		_, err = pr.Conn.Prepare(context.Background(), "slug-query", "INSERT INTO post_schema.post VALUES (default, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id")
		if err != nil {
			log.Println(err)
			return err
		}
	*/

	var pID int
	err = pr.Pool.QueryRow(
		context.Background(),
		"INSERT INTO post_schema.post VALUES (default, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id",
		p.Title, p.Slug+"-"+counter, p.Body, p.CreatedAt.UTC(), nil, p.Tags, p.Hidden, p.AuthorID, p.FeatureImgURL, p.Subtitle, p.Views,
	).Scan(&pID)

	if err != nil {
		log.Println(err)
		return err
	}

	p.Slug = p.Slug + "-" + counter

	p.ID = pID
	return nil
}

// FindByID returns the post with the given ID. Returns nil if the post doesn't exist or it's hidden in the database
func (pr *postRepository) FindByID(id int) (*models.Post, error) {
	post := models.Post{}

	err := pr.Pool.QueryRow(context.Background(),
		"SELECT * FROM post_schema.post WHERE NOT hidden AND id = $1", id,
	).Scan(&post.ID, &post.Title, &post.Slug, &post.Body, &post.CreatedAt, &post.UpdatedAt,
		&post.Tags, &post.Hidden, &post.AuthorID, &post.FeatureImgURL, &post.Subtitle, &post.Views,
	)

	if err != nil {
		return nil, err
	}
	post.Views++
	//pr.Conn.Prepare(context.Background(), "update-views-query", "UPDATE post_schema.post SET views=$1 WHERE id=$2")
	_, err = pr.Pool.Exec(context.Background(), "UPDATE post_schema.post SET views=$1 WHERE id=$2", post.Views, post.ID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &post, nil
}

// FindByIDAdmin returns the post with the given ID (including hidden). Returns nil if the post doesn't exist
func (pr *postRepository) FindByIDAdmin(id int) (*models.Post, error) {
	post := models.Post{}

	err := pr.Pool.QueryRow(context.Background(), "SELECT * FROM post_schema.post WHERE id = $1", id).Scan(
		&post.ID, &post.Title, &post.Slug, &post.Body, &post.CreatedAt, &post.UpdatedAt, &post.Tags,
		&post.Hidden, &post.AuthorID, &post.FeatureImgURL, &post.Subtitle, &post.Views,
	)

	if err != nil {
		return nil, err
	}

	return &post, nil
}

// Update updates the post with the given ID in the database
func (pr *postRepository) Update(p *models.Post) error {
	exists := pr.Exists(p.Slug)
	// Check if this is a new slug
	if !exists {
		//
		err := pr.updatePost(p)
		if err != nil {
			return err
		}

		return nil
	}

	// Post do exists
	// Now we want to find out if the slug is the post we are updating
	var postID int
	err := pr.Pool.QueryRow(context.Background(), "SELECT id FROM post_schema.post WHERE slug LIKE $1", p.Slug).Scan(&postID)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	if p.ID == postID {
		err := pr.updatePost(p)
		if err != nil {
			return err
		}

		return nil
	}

	// If its not the same post we append the next count number of that slug
	var slugCount int
	err = pr.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM post_schema.post where slug LIKE $1", "%"+p.Slug+"%").Scan(&slugCount)
	if err != nil {
		return err
	}
	counter := strconv.Itoa(slugCount + 1)
	p.Slug = p.Slug + "-" + counter

	err = pr.updatePost(p)
	if err != nil {
		return err
	}

	return nil
}

// updatePost is separated since it's used in multiple conditions in Update
func (pr *postRepository) updatePost(p *models.Post) error {
	_, err := pr.Pool.Exec(context.Background(), "UPDATE post_schema.post SET title=$1, slug=$2, body=$3, updated_at=$4, tags=$5, hidden=$6, feature_image_url=$7, subtitle=$8 WHERE id=$9", p.Title, p.Slug, p.Body, p.UpdatedAt, p.Tags, p.Hidden, p.FeatureImgURL, p.Subtitle, p.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// GetTotalPostCount returns the number of posts (including hidden) in the database
func (pr *postRepository) GetTotalPostCount() (int, error) {
	var count int
	err := pr.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM post_schema.post").Scan(&count)
	if err != nil {
		log.Println(err)
		return -1, err
	}

	return count, nil
}

// GetPublicPostCount returns the number of non-hidden posts in the database
func (pr *postRepository) GetPublicPostCount() (int, error) {
	var count int
	err := pr.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM post_schema.post WHERE NOT hidden").Scan(&count)
	if err != nil {
		log.Println(err)
		return -1, err
	}

	return count, nil
}

// FindBySlug returns the post with the given slug in
func (pr *postRepository) FindBySlug(slug string) (*models.Post, error) {
	post := models.Post{}

	err := pr.Pool.QueryRow(context.Background(), "SELECT * FROM post_schema.post WHERE NOT hidden AND slug LIKE $1", slug).Scan(
		&post.ID, &post.Title, &post.Slug, &post.Body, &post.CreatedAt, &post.UpdatedAt,
		&post.Tags, &post.Hidden, &post.AuthorID, &post.FeatureImgURL, &post.Subtitle, &post.Views,
	)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	post.Views++

	//pr.Conn.Prepare(context.Background(), "update-views-query", "UPDATE post_schema.post SET views=$1 WHERE slug LIKE $2")
	_, err = pr.Pool.Exec(context.Background(), "UPDATE post_schema.post SET views=$1 WHERE slug LIKE $2", post.Views, slug)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &post, nil
}

// Returns a single post matching the slug, including hidden posts. There should not be multiple posts with the same slug.
func (pr *postRepository) FindBySlugAdmin(slug string) (*models.Post, error) {
	post := models.Post{}
	err := pr.Pool.QueryRow(context.Background(), "SELECT * FROM post_schema.post WHERE slug LIKE $1", slug).Scan(
		&post.ID, &post.Title, &post.Slug, &post.Body, &post.CreatedAt, &post.UpdatedAt,
		&post.Tags, &post.Hidden, &post.AuthorID, &post.FeatureImgURL, &post.Subtitle, &post.Views,
	)

	if err != nil {
		log.Println(err)
		return nil, err
	}
	// Don't increment view count
	//err = pr.Conn.QueryRow(context.Background(), "UPDATE post_schema.post SET views=$1 WHERE slug LIKE $2", post.Views, slug).Scan()

	return &post, nil
}

// GetAll returns all posts (including hidden)
func (pr *postRepository) GetAll() ([]*models.Post, error) {
	var posts []*models.Post

	rows, err := pr.Pool.Query(context.Background(), "SELECT * FROM post_schema.post")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		p := new(models.Post)
		err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Body, &p.CreatedAt, &p.UpdatedAt, &p.Tags, &p.Hidden, &p.AuthorID, &p.FeatureImgURL, &p.Subtitle, &p.Views)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		posts = append(posts, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

// Paginate returns the keyset page of posts in the database
func (pr *postRepository) Paginate(maxID int, perPage int, tags []string) ([]*models.Post, int, error) {
	var posts []*models.Post

	var rows pgx.Rows
	var err error

	// For some reason, can't use same query w/ tags in latest pgx update
	if len(tags) == 0 {
		rows, err = pr.Pool.Query(context.Background(), "SELECT * FROM post_schema.post WHERE NOT hidden AND id < $1 ORDER BY created_at DESC, id DESC LIMIT $2", maxID, perPage)
	} else {
		rows, err = pr.Pool.Query(context.Background(), "SELECT * FROM post_schema.post WHERE NOT hidden AND id < $1 AND tags @> $2::text[] ORDER BY created_at DESC, id DESC LIMIT $3", maxID, tags, perPage)
	}
	defer rows.Close()
	if err != nil {
		log.Println(err)
		return nil, -1, err
	}
	var minID int
	for rows.Next() {
		p := new(models.Post)
		err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Body, &p.CreatedAt, &p.UpdatedAt, &p.Tags, &p.Hidden, &p.AuthorID, &p.FeatureImgURL, &p.Subtitle, &p.Views)

		if err != nil {
			log.Println(err)
			return nil, -1, err
		}

		// Limit p.Body to 250 characters
		limit := len(p.Body)
		if len(p.Body) > 250 {
			limit = 250
		}

		p.Body = p.Body[:limit]

		posts = append(posts, p)

		minID = p.ID
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil, -1, err
	}

	//var minID int
	//err = pr.QueryRow(context.Background(), "SELECT id FROM (SELECT * FROM post_schema.post WHERE id < $1 AND NOT hidden ORDER BY created_at DESC, id DESC LIMIT $2) AS trash_alias WHERE tags @> $3::text[] ORDER BY created_at LIMIT 1;", maxID, perPage, &tags).Scan(&minID)
	//if err != nil {
	//	log.Println(err)
	//	return nil, -1, err
	//}
	return posts, minID, nil
}

// Paginate returns the keyset page of posts (including hidden) in the database
func (pr *postRepository) PaginateAdmin(maxID int, perPage int, tags []string) ([]*models.Post, int, error) {
	var posts []*models.Post

	var rows pgx.Rows
	var err error

	// For some reason, can't use same query w/ tags in latest pgx update
	if len(tags) == 0 {
		rows, err = pr.Pool.Query(context.Background(), "SELECT * FROM post_schema.post WHERE id < $1 ORDER BY created_at DESC, id DESC LIMIT $2", maxID, perPage)
	} else {
		rows, err = pr.Pool.Query(context.Background(), "SELECT * FROM post_schema.post WHERE id < $1 AND tags @> $2::text[] ORDER BY created_at DESC, id DESC LIMIT $3", maxID, tags, perPage)
	}
	defer rows.Close()
	if err != nil {
		log.Println(err)
		return nil, -1, err
	}
	var minID int
	for rows.Next() {
		p := new(models.Post)
		err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Body, &p.CreatedAt, &p.UpdatedAt, &p.Tags, &p.Hidden, &p.AuthorID, &p.FeatureImgURL, &p.Subtitle, &p.Views)

		if err != nil {
			log.Println(err)
			return nil, -1, err
		}

		// Limit p.Body to 250 characters
		limit := len(p.Body)
		if len(p.Body) > 250 {
			limit = 250
		}

		p.Body = p.Body[:limit]

		posts = append(posts, p)

		minID = p.ID
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil, -1, err
	}

	return posts, minID, nil
}

// ResetSeq resets the post id sequence in the database
func (pr *postRepository) ResetSeq() error {
	row, err := pr.Pool.Query(context.Background(), "SELECT setval(pg_get_serial_sequence('post_schema.post', 'id'), coalesce(max(id),0)+ 1, false) FROM post_schema.post")

	if err != nil {
		log.Println(err)
		return err
	}
	defer row.Close()
	return nil
}

// GetLastID gets the last (highest) ID non-hidden post in the database
func (pr *postRepository) GetLastID() (int, error) {
	var lastID int
	err := pr.Pool.QueryRow(context.Background(), "SELECT id FROM post_schema.post WHERE NOT hidden ORDER BY created_at DESC LIMIT 1").Scan(&lastID)
	if err != nil {
		log.Println(err)
		return -1, err
	}

	return lastID, nil
}

// GetLastIDAdmin gets the last (highest) ID post (including hidden) in the database
func (pr *postRepository) GetLastIDAdmin() (int, error) {
	var lastID int
	err := pr.Pool.QueryRow(context.Background(), "SELECT id FROM post_schema.post ORDER BY created_at DESC LIMIT 1").Scan(&lastID)
	if err != nil {
		log.Println(err)
		return -1, err
	}

	return lastID, nil
}

// SearchQuery searches using title and tags. Returns results ordered by view count in descending order.
func (pr *postRepository) SearchQuery(title string, tags []string) ([]*models.Post, error) {
	var posts []*models.Post

	var rows pgx.Rows
	var err error

	// For some reason it needs a separate query for tags to return rows
	if len(tags) == 0 {
		rows, err = pr.Pool.Query(context.Background(),
			"SELECT * FROM post_schema.post WHERE NOT hidden AND LOWER(title) LIKE LOWER('%' || $1 || '%') ORDER BY views DESC LIMIT 5",
			title,
		)
	} else {
		rows, err = pr.Pool.Query(context.Background(),
			"SELECT * FROM post_schema.post WHERE NOT hidden AND LOWER(title) LIKE LOWER('%' || $1 || '%') AND tags @> $2 ORDER BY views DESC LIMIT 5",
			title, tags,
		)
	}

	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		p := new(models.Post)
		err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Body, &p.CreatedAt, &p.UpdatedAt, &p.Tags, &p.Hidden, &p.AuthorID, &p.FeatureImgURL, &p.Subtitle, &p.Views)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		// Limit p.Body to 250 characters
		limit := len(p.Body)
		if len(p.Body) > 250 {
			limit = 250
		}

		p.Body = p.Body[:limit]

		posts = append(posts, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}
