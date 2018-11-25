package database

import (
	"github.com/lib/pq"

	db "github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"

	"chat/models"
)

func CreateMessage(m *models.Message) (*models.Message, error) {
	res := &models.Message{}
	qres := db.DB().QueryRowx(`
		INSERT INTO message (author_id, to_user, message_text)
		VALUES ($1, $2, $3) RETURNING *`,
		m.Author, m.To, m.Message)
	if err := qres.Err(); err != nil {
		pqErr := err.(*pq.Error)
		switch pqErr.Code {
		case "23502":
			return res, db.ErrNotNullConstraintViolation
		}
	}
	err := qres.StructScan(res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func GetAllGlobalMessages() (*[]models.Message, error) {
	res := &[]models.Message{}
	err := db.DB().Select(res, `
		SELECT * FROM message
		WHERE to_user IS NULL
		ORDER BY created`)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func EditMessage(m *models.Message) error {

	return nil
}

func DeleteMessage(m *models.Message) error {

	return nil
}
