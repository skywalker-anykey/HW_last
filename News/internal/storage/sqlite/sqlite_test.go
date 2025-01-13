package sqlite

import (
	"testing"
)

const storagePath = "../../../cmd/NewsServer/News.db"
const RealPostId = "https://habr.com/ru/articles/871918/"

func TestNew(t *testing.T) {
	s, err := New(storagePath)

	if err != nil {
		t.Error(err)
	}

	if s == nil {
		t.Error("New returned nil")
	}

}

func TestStorage_Count(t *testing.T) {
	s, _ := New(storagePath)

	t.Log(s.Count())
}

func TestStorage_Posts(t *testing.T) {
	s, _ := New(storagePath)
	rows, _ := s.Posts(5, 10)
	for _, v := range rows {
		t.Log("ID:", v.ID, "Title:", v.Title)
	}
}

func TestStorage_PostById(t *testing.T) {
	s, _ := New(storagePath)
	p, err := s.PostById(RealPostId)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(p)
	}
}

func TestStorage_Filter(t *testing.T) {
	s, _ := New(storagePath)
	rows, err := s.Filter("go", 0, 28)
	if err != nil {
		t.Error(err)
	}
	for _, v := range rows {
		t.Log("ID:", v.ID, "Title:", v.Title)
	}
}

func TestStorage_CountOfFilter(t *testing.T) {
	s, _ := New(storagePath)
	c, err := s.CountOfFilter("go")
	if err != nil {
		t.Error(err)
	}
	t.Log("count:", c)
}
