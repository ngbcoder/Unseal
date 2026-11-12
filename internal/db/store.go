package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ngbcoder/Unseal/internal/types"
	"go.etcd.io/bbolt"
)

// The bucket names.
var (
	bucketMeta     = []byte("meta")
	bucketProfiles = []byte("profiles")
	bucketAccounts = []byte("accounts")
	bucketBooks    = []byte("books")
)

// Store wraps a bbolt database.
//
// The database is a single file. The buckets are:
//
//	meta/          format version and small pieces of global state
//	profiles/      registered device profiles
//	accounts/      one sub-bucket per account and region
//
// Each account sub-bucket holds a books/ sub-bucket, keyed by ASIN.
type Store struct {
	db   *bbolt.DB
	path string
}

// Open opens (or creates) the database at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("db: cannot create the data directory: %w", err)
	}

	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("db: cannot open %s: %w", path, err)
	}

	s := &Store{db: db, path: path}
	if err := s.ensureBuckets(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Path returns the path of the database file.
func (s *Store) Path() string { return s.path }

// DB returns the underlying bbolt database, for callers that need a
// transaction that spans several operations.
func (s *Store) DB() *bbolt.DB { return s.db }

// ensureBuckets creates the top-level buckets.
func (s *Store) ensureBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		for _, name := range [][]byte{bucketMeta, bucketProfiles, bucketAccounts} {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return fmt.Errorf("db: cannot create bucket %s: %w", name, err)
			}
		}
		return nil
	})
}

// accountBucket returns the bucket for an account and region, creating it
// when it does not exist.
func (s *Store) accountBucket(tx *bbolt.Tx, accountID string, region types.Region) (*bbolt.Bucket, error) {
	accounts := tx.Bucket(bucketAccounts)
	if accounts == nil {
		return nil, errors.New("db: the accounts bucket is missing")
	}

	key := []byte(accountID + "-" + string(region))
	acct, err := accounts.CreateBucketIfNotExists(key)
	if err != nil {
		return nil, fmt.Errorf("db: cannot create the account bucket: %w", err)
	}

	if _, err := acct.CreateBucketIfNotExists(bucketBooks); err != nil {
		return nil, fmt.Errorf("db: cannot create the books bucket: %w", err)
	}

	return acct, nil
}

// accountBucketReadOnly returns the bucket for an account and region, or
// nil when it does not exist.
func (s *Store) accountBucketReadOnly(tx *bbolt.Tx, accountID string, region types.Region) *bbolt.Bucket {
	accounts := tx.Bucket(bucketAccounts)
	if accounts == nil {
		return nil
	}
	return accounts.Bucket([]byte(accountID + "-" + string(region)))
}

// booksBucket returns the books bucket for an account and region.
func (s *Store) booksBucket(tx *bbolt.Tx, accountID string, region types.Region, create bool) (*bbolt.Bucket, error) {
	if create {
		acct, err := s.accountBucket(tx, accountID, region)
		if err != nil {
			return nil, err
		}
		return acct.Bucket(bucketBooks), nil
	}

	acct := s.accountBucketReadOnly(tx, accountID, region)
	if acct == nil {
		return nil, nil
	}
	return acct.Bucket(bucketBooks), nil
}

// GetBook returns the book with the given ASIN, or nil when it does not
// exist.
func (s *Store) GetBook(accountID string, region types.Region, asin string) (*BookDoc, error) {
	var book *BookDoc

	err := s.db.View(func(tx *bbolt.Tx) error {
		books, err := s.booksBucket(tx, accountID, region, false)
		if err != nil {
			return err
		}
		if books == nil {
			return nil
		}
		data := books.Get([]byte(asin))
		if data == nil {
			return nil
		}
		var b BookDoc
		if err := json.Unmarshal(data, &b); err != nil {
			return fmt.Errorf("db: cannot decode the book %s: %w", asin, err)
		}
		book = &b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return book, nil
}

// ListBooks returns every book of an account and region.
//
// The order is the order in which the books were written, which is the
// order in which the library listing returns them.
func (s *Store) ListBooks(accountID string, region types.Region) ([]*BookDoc, error) {
	var books []*BookDoc

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket, err := s.booksBucket(tx, accountID, region, false)
		if err != nil {
			return err
		}
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var b BookDoc
			if err := json.Unmarshal(v, &b); err != nil {
				return fmt.Errorf("db: cannot decode a book: %w", err)
			}
			books = append(books, &b)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return books, nil
}

// PutBook stores a book, replacing any existing entry with the same ASIN.
func (s *Store) PutBook(accountID string, region types.Region, book *BookDoc) error {
	if book == nil || book.Asin == "" {
		return errors.New("db: a book needs an ASIN")
	}
	if book.V == 0 {
		book.V = DocumentVersion
	}

	data, err := json.Marshal(book)
	if err != nil {
		return fmt.Errorf("db: cannot encode the book %s: %w", book.Asin, err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		books, err := s.booksBucket(tx, accountID, region, true)
		if err != nil {
			return err
		}
		return books.Put([]byte(book.Asin), data)
	})
}

// DeleteBook removes a book.
func (s *Store) DeleteBook(accountID string, region types.Region, asin string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		books, err := s.booksBucket(tx, accountID, region, false)
		if err != nil {
			return err
		}
		if books == nil {
			return nil
		}
		return books.Delete([]byte(asin))
	})
}

// ListConversions returns every conversion of an account and region.
//
// A conversion is extracted from the book or component that owns it, so
// the list is derived rather than stored separately. The whole-book
// conversion comes first, then the parts in order.
func (s *Store) ListConversions(accountID string, region types.Region) ([]*ConversionDoc, error) {
	books, err := s.ListBooks(accountID, region)
	if err != nil {
		return nil, err
	}

	var out []*ConversionDoc
	for _, b := range books {
		out = append(out, b.Conversions()...)
	}
	return out, nil
}

// UpdateWholeConversion replaces the whole-book conversion of a book.
func (s *Store) UpdateWholeConversion(accountID string, region types.Region, asin string, conv *ConversionDoc) error {
	return s.updateBook(accountID, region, asin, func(b *BookDoc) {
		b.WholeConversion = conv
	})
}

// UpdateComponentConversion replaces the conversion of one part of a book.
//
// The index is the position of the component in the book's component list.
func (s *Store) UpdateComponentConversion(accountID string, region types.Region, asin string, index int, conv *ConversionDoc) error {
	return s.updateBook(accountID, region, asin, func(b *BookDoc) {
		if index < 0 || index >= len(b.Components) {
			return
		}
		b.Components[index].Conversion = conv
	})
}

// UpdateLicense replaces the license key and IV of a book or one of its
// parts.
//
// A negative index addresses the whole book; a non-negative index addresses
// the part at that position.
func (s *Store) UpdateLicense(accountID string, region types.Region, asin string, index int, key, iv string) error {
	return s.updateBook(accountID, region, asin, func(b *BookDoc) {
		if index < 0 {
			b.LicenseKey = key
			b.LicenseIv = iv
			return
		}
		if index < len(b.Components) {
			b.Components[index].LicenseKey = key
			b.Components[index].LicenseIv = iv
		}
	})
}

// UpdateDownloadFileName stores the file name of a downloaded payload.
//
// A negative index addresses the whole book.
func (s *Store) UpdateDownloadFileName(accountID string, region types.Region, asin string, index int, name string) error {
	return s.updateBook(accountID, region, asin, func(b *BookDoc) {
		if index < 0 {
			if b.WholeConversion != nil {
				b.WholeConversion.DownloadFileName = name
			}
			return
		}
		if index < len(b.Components) && b.Components[index].Conversion != nil {
			b.Components[index].Conversion.DownloadFileName = name
		}
	})
}

// SetDeleted marks a book as removed from the account's library, or
// unmarks it when it comes back.
func (s *Store) SetDeleted(accountID string, region types.Region, asin string, deleted bool) error {
	return s.updateBook(accountID, region, asin, func(b *BookDoc) {
		b.Deleted = deleted
	})
}

// updateBook loads a book, applies fn to it, and stores it back.
func (s *Store) updateBook(accountID string, region types.Region, asin string, fn func(*BookDoc)) error {
	book, err := s.GetBook(accountID, region, asin)
	if err != nil {
		return err
	}
	if book == nil {
		return fmt.Errorf("db: no book %s", asin)
	}
	fn(book)
	return s.PutBook(accountID, region, book)
}

// Meta returns a value from the meta bucket.
func (s *Store) Meta(key string) (string, error) {
	var value string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMeta)
		if b == nil {
			return nil
		}
		if v := b.Get([]byte(key)); v != nil {
			value = string(v)
		}
		return nil
	})
	return value, err
}

// SetMeta stores a value in the meta bucket.
func (s *Store) SetMeta(key, value string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMeta)
		if b == nil {
			return errors.New("db: the meta bucket is missing")
		}
		return b.Put([]byte(key), []byte(value))
	})
}

// ListAccountKeys returns the keys of every account and region that has
// at least one book.
func (s *Store) ListAccountKeys() ([]string, error) {
	var keys []string
	err := s.db.View(func(tx *bbolt.Tx) error {
		accounts := tx.Bucket(bucketAccounts)
		if accounts == nil {
			return nil
		}
		return accounts.ForEach(func(k, _ []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	return keys, err
}

// HardDelete removes every book of an account and region.
//
// It is the Go equivalent of deleting the whole account bucket. It is
// used by the tests and by the "forget this account" command.
func (s *Store) HardDelete(accountID string, region types.Region) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		accounts := tx.Bucket(bucketAccounts)
		if accounts == nil {
			return nil
		}
		return accounts.DeleteBucket([]byte(accountID + "-" + string(region)))
	})
}
