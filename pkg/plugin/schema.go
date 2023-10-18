package plugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type schemaInferenceState struct {
	typeGuesses map[string]data.FieldType
	currentRow  map[string]data.FieldType
	ignored     map[string]struct{}
	afterFirst  bool
}

func NewSchemaInference(ignored map[string]struct{}) schemaInferenceState {
	return schemaInferenceState{
		typeGuesses: make(map[string]data.FieldType),
		currentRow:  make(map[string]data.FieldType),
		ignored:     ignored,
		afterFirst:  false,
	}
}

func (s *schemaInferenceState) updateField(name string, value interface{}) error {
	if _, ignored := s.ignored[name]; ignored {
		return nil
	}
	_, guess, err := ToGrafanaValue(value)
	if err != nil {
		return err
	}
	// if err is nil and type is still unknown, original value must be null or undefined,
	// which we treat the same as being absent
	if guess == data.FieldTypeUnknown {
		return nil
	}
	s.currentRow[name] = guess
	return nil
}

func (s *schemaInferenceState) updateDoc(doc timestepDocument) error {
	for name, value := range doc {
		err := s.updateField(name, value)
		if err != nil {
			return err
		}
	}
	return s.nextDoc()
}

func (s *schemaInferenceState) nextDoc() error {
	mismatchOld := make(map[string]data.FieldType, len(s.typeGuesses))
	mismatchNew := make(map[string]data.FieldType, len(s.typeGuesses))
	if s.afterFirst {
		for name, guess := range s.typeGuesses {
			// Any non-nullable guesses that aren't in the current row must be nullable
			_, stillHere := s.currentRow[name]
			if !stillHere && !guess.Nullable() {
				log.DefaultLogger.Debug("New field discovered after first row, making type nullable", "name", name, "row", s.typeGuesses, "guesses", s.typeGuesses)
				s.typeGuesses[name] = guess.NullableType()
			}
		}
	}
	for name, currentType := range s.currentRow {
		guess, known := s.typeGuesses[name]
		if !s.afterFirst {
			// If this is the first document, guesses /are/ the current row
			s.typeGuesses[name] = currentType
		} else if s.afterFirst && !known && !currentType.Nullable() {
			// If a new field is found after the first document,
			// the old guess must be made nullable
			log.DefaultLogger.Debug("Field missing after first row, making type nullable", "name", name, "row", s.typeGuesses, "guesses", s.typeGuesses)
			s.typeGuesses[name] = currentType.NullableType()
		} else if s.afterFirst && known && (currentType == guess || currentType.NullableType() == guess) {
			// If the new guess and the old guess are the same modulo nullability,
			// then no action is necessary
		} else {
			// Otherwise, record a mismatch
			// We don't exit immediately so that the user can know about and fix all mismatches at once
			mismatchOld[name] = guess
			mismatchNew[name] = currentType
		}
	}

	if len(mismatchOld) != 0 {
		errMsg := strings.Builder{}
		errMsg.WriteString("Field(s) appeared with different types. Please ensure value fields are the same type in each document: ")
		first := false
		for name, old := range mismatchOld {
			new := mismatchNew[name]
			if first {
				errMsg.WriteString(",")
			} else {
				first = true
			}
			errMsg.WriteString(fmt.Sprintf("%s (%s vs %s)", name, old, new))
		}
		return errors.New(errMsg.String())
	}

	s.afterFirst = true
	s.currentRow = make(map[string]data.FieldType, len(s.typeGuesses))
	return nil
}

func (s *schemaInferenceState) finish() []field {
	fields := make([]field, 0, len(s.typeGuesses))
	for name, guess := range s.typeGuesses {
		fields = append(fields, field{Name: name, Type: guess})
	}
	return fields
}
