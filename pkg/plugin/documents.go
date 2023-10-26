package plugin

import (
	"context"
	"fmt"
	"runtime"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
)

type frameCountDocument struct {
	Labels map[string]interface{} `bson:"_id"`
	Count  int                    `bson:"count"`
}

type timestepDocument = map[string]interface{}

type field struct {
	Name string
	Type data.FieldType
}

func (f *field) get(doc timestepDocument) interface{} {
	return doc[f.Name]
}

type resultParser struct {
	frames map[string]*data.Frame
	model  resolvedQueryModel
}

func (p *resultParser) parseQueryResultDocument(doc timestepDocument) (err error) {
	defer func() {
		if panic_ := recover(); panic_ != nil {
			buf := make([]byte, 1<<16)
			buflen := runtime.Stack(buf, false)
			log.DefaultLogger.Error("Panic while parsing document", "document", doc, "error", panic_, "trace", string(buf[:buflen]))
			switch panic_.(type) {
			case error:
				err = panic_.(error)
			default:
				err = fmt.Errorf("%v", panic_)
			}
		}
	}()
	labels, labelsID := p.model.getLabels(doc)
	frame, ok := p.frames[labelsID]
	if !ok {
		log.DefaultLogger.Debug("Creating frame for unique label combination", "doc", doc, "labels", labels, "labelsID", labelsID)
		frame, err = p.model.makeFrame(labelsID, labels)
		if err != nil {
			return err
		}
		p.frames[labelsID] = frame
	}
	row, err := p.model.getValues(doc)
	if err != nil {
		return errors.Wrap(err, "Failed to extract value columns")
	}
	log.DefaultLogger.Debug("Parsed row", "row", row, "id", labelsID)
	frame.AppendRow(row...)

	return nil
}

type bufferingCursor struct {
	*mongo.Cursor
	buffer []timestepDocument
}

func (c *bufferingCursor) Next(ctx context.Context) (doc timestepDocument, more bool, err error) {
	more = c.Cursor.Next(ctx)
	if !more {
		err = c.Cursor.Err()
		return
	}

	doc = make(timestepDocument)
	err = c.Cursor.Decode(&doc)
	if err != nil {
		more = false
		return
	}

	c.buffer = append(c.buffer, doc)
	more = true
	return
}

type bufferedCursor struct {
	*mongo.Cursor
	buffer []timestepDocument
}

func (c *bufferedCursor) Next(ctx context.Context) (doc timestepDocument, more bool, decodeErr bool, err error) {
	if len(c.buffer) != 0 {
		doc = c.buffer[0]
		c.buffer = c.buffer[1:]
		more = true
		return
	}

	more = c.Cursor.Next(ctx)
	if !more {
		err = c.Cursor.Err()
		return
	}

	doc = make(timestepDocument)
	err = c.Decode(&doc)
	if err != nil {
		decodeErr = true
		more = false
	}
	return
}
