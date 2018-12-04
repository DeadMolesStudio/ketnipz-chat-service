package database

import (
	"github.com/lib/pq"

	db "github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"

	"chat/models"
)

func CreateMessage(dm *db.DatabaseManager, m *models.Message) (*models.Message, error) {
	dbo, err := dm.DB()
	if err != nil {
		return nil, err
	}
	qres := dbo.QueryRowx(`
	INSERT INTO message (author_id, to_user, message_text)
	VALUES ($1, $2, $3) RETURNING *`,
		m.Author, m.To, m.Message)
	if err := qres.Err(); err != nil {
		pqErr := err.(*pq.Error)
		switch pqErr.Code {
		case "23502":
			return nil, db.ErrNotNullConstraintViolation
		}
	}
	res := &models.Message{}
	err = qres.StructScan(res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func GetAllGlobalMessages(dm *db.DatabaseManager) (*[]models.Message, error) {
	dbo, err := dm.DB()
	if err != nil {
		return nil, err
	}
	res := &[]models.Message{}
	err = dbo.Select(res, `
		SELECT * FROM message
		WHERE to_user IS NULL
		ORDER BY created`)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func EditMessage(dm *db.DatabaseManager, m *models.Message) error {

	return nil
}

func DeleteMessage(dm *db.DatabaseManager, m *models.Message) error {

	return nil
}
