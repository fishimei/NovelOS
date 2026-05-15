package models

import (
	"database/sql/driver"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type JSONB []byte

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONB) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*j = JSONB("null")
		return nil
	case []byte:
		*j = append((*j)[:0], v...)
		return nil
	case string:
		*j = JSONB(v)
		return nil
	default:
		return fmt.Errorf("unsupported jsonb scan type %T", value)
	}
}

func (JSONB) GormDataType() string {
	return "json"
}

func (JSONB) GormDBDataType(*gorm.DB, *schema.Field) string {
	return "jsonb"
}
