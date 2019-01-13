package panel

import (
	"github.com/jroimartin/gocui"
	"github.com/skanehira/docui/common"
)

type TaskList struct {
	*Gui
	name string
	Position
	Tasks []Task
}

type Task struct {
	Name    string `tag:"NAME" len:"min:0.3 max:0.3"`
	Status  string `tag:"STATUS" len:"min:0.3 max:0.3"`
	Created string `tag:"CREATED" len:"min:0.3 max:0.3"`
}

func NewTaskList(gui *Gui, name string, x, y, w, h int) *TaskList {
	return &TaskList{
		Gui:  gui,
		name: name,
		Position: Position{
			x: x,
			y: y,
			w: w,
			h: h,
		},
	}
}

func (t *TaskList) SetView(g *gocui.Gui) error {
	// set header panel
	if v, err := g.SetView(TaskListHeaderPanel, t.x, t.y, t.w, t.h); err != nil {
		if err != gocui.ErrUnknownView {
			panic(err)
		}

		v.Wrap = true
		v.Frame = true
		v.Title = v.Name()
		v.FgColor = gocui.AttrBold | gocui.ColorWhite
		common.OutputFormatedHeader(v, &Task{})
	}

	// set scroll panel
	v, err := g.SetView(t.name, t.x, t.y+1, t.w, t.h)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.Wrap = true
		v.FgColor = gocui.ColorCyan
		v.SelBgColor = gocui.ColorWhite
		v.SelFgColor = gocui.ColorBlack | gocui.AttrBold
		v.SetOrigin(0, 0)
		v.SetCursor(0, 0)
	}

	t.SetKeyBinding()

	return nil
}

func (t *TaskList) Name() string {
	return t.name
}

func (t *TaskList) Refresh(g *gocui.Gui, v *gocui.View) error {
	// do nothing
	return nil
}

func (t *TaskList) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case ch != 0 && mod == 0:
		v.EditWrite(ch)
	case key == gocui.KeySpace:
		v.EditWrite(' ')
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
		return
	case key == gocui.KeyArrowRight:
		v.MoveCursor(+1, 0, false)
		return
	}
}

func (t *TaskList) SetKeyBinding() {
	t.SetKeyBindingToPanel(t.name)
}
