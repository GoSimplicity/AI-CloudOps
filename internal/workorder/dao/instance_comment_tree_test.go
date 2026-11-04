package dao

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type commentTreeEdge struct {
	id, parent, order int
	status            int8
}

func commentTreeFixture(t *testing.T, edges []commentTreeEdge) (*workorderInstanceCommentDAO, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "comment-tree.sqlite")), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.WorkorderInstanceComment{}); err != nil {
		t.Fatal(err)
	}
	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, edge := range edges {
		status := edge.status
		if status == 0 {
			status = model.CommentStatusNormal
		}
		row := model.WorkorderInstanceComment{Model: model.Model{ID: edge.id, CreatedAt: origin.Add(time.Duration(edge.order) * time.Second)}, InstanceID: 7, OperatorID: 7, OperatorName: "trusted", Content: fmt.Sprintf("comment-%d", edge.id), Type: model.CommentTypeNormal, Status: status, IsSystem: 2}
		if edge.parent != 0 {
			parent := edge.parent
			row.ParentID = &parent
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	return NewWorkorderInstanceCommentDAO(db, zap.NewNop()).(*workorderInstanceCommentDAO), db
}
func commentTreeIDs(roots []*model.WorkorderInstanceComment) []int {
	result := []int{}
	queue := append([]*model.WorkorderInstanceComment(nil), roots...)
	for head := 0; head < len(queue); head++ {
		node := queue[head]
		result = append(result, node.ID)
		for i := range node.Children {
			queue = append(queue, &node.Children[i])
		}
	}
	return result
}
func TestCommentTreeRetainsEveryLevelRegardlessOfInputOrder(t *testing.T) {
	for _, depth := range []int{3, 5} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("depth%d/reverse%v", depth, reverse), func(t *testing.T) {
				edges := []commentTreeEdge{}
				for id := 1; id <= depth; id++ {
					parent := id - 1
					order := id
					if reverse {
						order = depth - id
					}
					edges = append(edges, commentTreeEdge{id: id, parent: parent, order: order})
				}
				dao, db := commentTreeFixture(t, edges)
				tree, err := dao.GetInstanceCommentsTree(context.Background(), 7)
				if err != nil {
					t.Fatal(err)
				}
				if len(tree) != 1 {
					t.Fatal("chain lost its root")
				}
				current := tree[0]
				for id := 1; id <= depth; id++ {
					if current.ID != id {
						t.Fatal("chain changed parent linkage")
					}
					if id < depth {
						if len(current.Children) != 1 {
							t.Fatalf("comment %d lost its descendant at depth %d", id, id+1)
						}
						current = &current.Children[0]
					} else if len(current.Children) != 0 {
						t.Fatal("leaf gained children")
					}
				}
				var count int64
				if db.Model(&model.WorkorderInstanceComment{}).Count(&count).Error != nil || count != int64(depth) {
					t.Fatal("tree read changed stored comments")
				}
			})
		}
	}
}
func TestCommentTreePreservesRootsSiblingsAndMissingParentPolicy(t *testing.T) {
	edges := []commentTreeEdge{{id: 20, order: 1}, {id: 12, parent: 10, order: 2}, {id: 11, parent: 10, order: 3}, {id: 10, order: 4}, {id: 30, parent: 999, order: 5}, {id: 31, parent: 30, order: 6}, {id: 40, order: 7, status: model.CommentStatusHidden}, {id: 41, parent: 40, order: 8}, {id: 50, order: 9, status: model.CommentStatusDeleted}}
	dao, db := commentTreeFixture(t, edges)
	tree, err := dao.GetInstanceCommentsTree(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 2 || tree[0].ID != 20 || tree[1].ID != 10 || len(tree[1].Children) != 2 || tree[1].Children[0].ID != 12 || tree[1].Children[1].ID != 11 {
		t.Fatal("input root/sibling order or existing orphan/normal-status policy changed")
	}
	var count int64
	db.Model(&model.WorkorderInstanceComment{}).Count(&count)
	if count != int64(len(edges)) {
		t.Fatal("hidden/orphan rows were removed")
	}
}

func TestCommentTreeDeepChainIsIterativeAndLeavesInputUntouched(t *testing.T) {
	const depth = 10000
	input := make([]*model.WorkorderInstanceComment, 0, depth)
	for id := depth; id >= 1; id-- {
		row := &model.WorkorderInstanceComment{Model: model.Model{ID: id}, InstanceID: 7, Content: "deep-chain"}
		if id > 1 {
			parent := id - 1
			row.ParentID = &parent
		}
		input = append(input, row)
	}
	input[len(input)-1].Children = []model.WorkorderInstanceComment{{Model: model.Model{ID: 99000}}}
	tree, err := (&workorderInstanceCommentDAO{}).buildCommentTree(input)
	if err != nil || len(tree) != 1 {
		t.Fatal("deep chain construction failed", err)
	}
	node := tree[0]
	for id := 1; id <= depth; id++ {
		if node.ID != id {
			t.Fatal("deep chain lost a level")
		}
		if id < depth {
			if len(node.Children) != 1 {
				t.Fatal("deep chain was truncated")
			}
			node = &node.Children[0]
		}
	}
	if len(input[0].Children) != 0 || len(input[len(input)-1].Children) != 1 || input[len(input)-1].Children[0].ID != 99000 {
		t.Fatal("tree construction mutated its source comment objects")
	}
}

func TestCommentTreeInvalidGraphsFailWithoutPartialHistoryOrDataWrites(t *testing.T) {
	for _, edges := range [][]commentTreeEdge{
		{{id: 1, parent: 1, order: 1}},
		{{id: 1, order: 1}, {id: 2, parent: 3, order: 2}, {id: 3, parent: 2, order: 3}},
	} {
		t.Run(fmt.Sprint(edges), func(t *testing.T) {
			dao, db := commentTreeFixture(t, edges)
			tree, err := dao.GetInstanceCommentsTree(context.Background(), 7)
			if err == nil || tree != nil {
				t.Fatal("cyclic stored graph was returned as apparently complete partial history")
			}
			flat, err := dao.GetInstanceComments(context.Background(), 7)
			if err != nil || len(flat) != len(edges) {
				t.Fatal("invalid graph read removed or hid flat stored rows")
			}
			var count int64
			db.Model(&model.WorkorderInstanceComment{}).Count(&count)
			if count != int64(len(edges)) {
				t.Fatal("read mutated corrupt stored history")
			}
		})
	}
	for _, rows := range [][]*model.WorkorderInstanceComment{
		{nil},
		{{Model: model.Model{ID: 1}}, {Model: model.Model{ID: 1}}},
	} {
		if tree, err := (&workorderInstanceCommentDAO{}).buildCommentTree(rows); err == nil || tree != nil {
			t.Fatal("invalid node identity was silently chosen or discarded")
		}
	}
}
func TestCommentTreeEqualTimestampsUseStableIDs(t *testing.T) {
	dao, _ := commentTreeFixture(t, []commentTreeEdge{{id: 30, parent: 10}, {id: 10}, {id: 20, parent: 10}, {id: 1}})
	for attempt := 0; attempt < 2; attempt++ {
		tree, err := dao.GetInstanceCommentsTree(context.Background(), 7)
		if err != nil || len(tree) != 2 || tree[0].ID != 1 || tree[1].ID != 10 || len(tree[1].Children) != 2 || tree[1].Children[0].ID != 20 || tree[1].Children[1].ID != 30 {
			t.Fatal("equal-time roots or siblings were not deterministically ordered")
		}
	}
}

func TestCommentTreeAllOrphansKeepNilRootContract(t *testing.T) {
	parent := 999
	rows := []*model.WorkorderInstanceComment{{Model: model.Model{ID: 1}, ParentID: &parent}}
	tree, err := (&workorderInstanceCommentDAO{}).buildCommentTree(rows)
	if err != nil || tree != nil {
		t.Fatal("orphan-only roots changed from the existing nil/null contract")
	}
}
