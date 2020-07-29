// generic crud code
// do not edit. only generic code in this file. all customizations done in seperate methods in other go files

package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/muly/go-common/errors"
	"google.golang.org/api/iterator"
)

// Post posts the given list of records into the database collection
// returns list of errors (in the format errors.ErrMsgs) for all the failed records
func Post(ctx context.Context, dbConn *firestore.Client, list []Collection) error {
	var errs errors.ErrMsgs
	for _, r := range list {
		r.CreatedDate = time.Now()
		r.LastUpdate = time.Now()
		log.Printf("POST CRUD")
		_, err := dbConn.Collection(CollectionName).Doc(r.ID()).Create(ctx, r)
		if err != nil {
			log.Printf("POST CRUD error: %#v", err)
			errType := errors.ErrGeneric
			if strings.Contains(err.Error(), "code = AlreadyExists desc = Document already exists") {
				errType = errors.ErrExists
			}
			errs = append(errs, errors.NewError(errType, r.ID()).(errors.ErrMsg))
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Put updates the record
// if unique fields are missing in the input struct, return error
// matches the record based on the db id and updates the rest of the field with what is provided in the input
func Put(ctx context.Context, dbConn *firestore.Client, r Collection) error {
	if r.ID() == "" {
		return fmt.Errorf("key fields are missing: key %s", r.ID())
	}
	if !exists(ctx, dbConn, r.ID()) {
		return fmt.Errorf("document does not exists to update: key %s", r.ID())
	}

	r.LastUpdate = time.Now()
	_, err := dbConn.Collection(CollectionName).Doc(r.ID()).Set(ctx, r)
	if err != nil {
		return err
	}

	return nil
}

// Delete deletes the record
// if db id is blank in the input, returns generic error
// if db id is not found in the database, returns not found error
// matches the record based on the db id and delete the record
func Delete(ctx context.Context, dbConn *firestore.Client, dbID string) error {
	if dbID == "" {
		return errors.NewError(errors.ErrEmptyInput, "dbID")
	}
	if !exists(ctx, dbConn, dbID) {
		return errors.NewError(errors.ErrNotFound, dbID)
	}

	_, err := dbConn.Collection(CollectionName).Doc(dbID).Delete(ctx)
	if err != nil {
		return errors.NewError(errors.ErrGeneric, err.Error())
	}

	return nil
}

// GetByID gets the record based on the db id provided
// if db id is blank in the input, return error
// if record is not found, error is returned
// Note: unlike Query(), Get doesn't apply Valid=True filter
func GetByID(ctx context.Context, dbConn *firestore.Client, dbID string) (Collection, error) {
	if dbID == "" {
		return Collection{}, fmt.Errorf("dbid is missing, provide id")
	}

	r, err := dbConn.Collection(CollectionName).Doc(dbID).Get(ctx)
	if err != nil {
		return Collection{}, err
	}
	v := Collection{}
	r.DataTo(&v)

	return v, nil
}

// Get gets the records based on the keys and values provided
func Get(ctx context.Context, dbConn *firestore.Client, field string, fieldValue string) ([]Collection, error) {
	if field == "" {
		return []Collection{}, fmt.Errorf("field is missing, provide field")
	}

	vOne := Collection{}
	v := []Collection{}

	iter := dbConn.Collection(CollectionName).Where(field, "==", fieldValue).Documents(ctx)
	for {
		doc, err := iter.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return []Collection{}, err
		}

		out, err := json.Marshal(doc.Data())
		if err != nil {
			return []Collection{}, err
		}
		err = json.Unmarshal(out, &vOne)
		if err != nil {
			return []Collection{}, err
		}

		v = append(v, vOne)
	}
	return v, nil
}

func exists(ctx context.Context, dbConn *firestore.Client, dbID string) bool {
	_, err := dbConn.Collection(CollectionName).Doc(dbID).Get(ctx)
	if err != nil {
		return false
	}
	return true
}
