package sqlite

import (
	"APIGateway/Comments/internal/storage"
	"testing"
)

const storagePath = "../../../cmd/CommentsServer/Comments.db"
const RealNewsId = "Test_News_ID1"

func TestNew(t *testing.T) {
	s, err := New(storagePath)

	if err != nil {
		t.Error(err)
	}

	if s == nil {
		t.Error("New returned nil")
	}
}

func TestStorage_AddComment(t *testing.T) {
	s, _ := New(storagePath)
	c := storage.Comment{
		NewsId:  RealNewsId,
		Content: "Test_News_ID1Content3",
	}
	t.Log("Adding new comment:", c)
	err := s.AddComment(c)
	if err != nil {
		t.Error(err)
	}
}

func TestStorage_Comments(t *testing.T) {
	s, _ := New(storagePath)
	c, err := s.Comments(RealNewsId)
	if err != nil {
		t.Error(err)
	}
	t.Log(c)
}
