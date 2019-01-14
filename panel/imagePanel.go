package panel

import (
	"fmt"
	"os"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/jroimartin/gocui"
	"github.com/skanehira/docui/common"
)

const (
	VolumeTypeBind   = "bind"
	VolumeTypeVolume = "volume"
)

type ImageList struct {
	*Gui
	name string
	Position
	Images []*Image
	Data   map[string]interface{}
	filter string
	form   *Form
}

type Image struct {
	ID      string `tag:"ID" len:"min:0.1 max:0.2"`
	Repo    string `tag:"REPOSITORY" len:"min:0.1 max:0.3"`
	Tag     string `tag:"TAG" len:"min:0.1 max:0.1"`
	Created string `tag:"CREATED" len:"min:0.1 max:0.2"`
	Size    string `tag:"SIZE" len:"min:0.1 max:0.2"`
}

func NewImageList(gui *Gui, name string, x, y, w, h int) *ImageList {
	i := &ImageList{
		Gui:      gui,
		name:     name,
		Position: Position{x, y, w, h},
		Data:     make(map[string]interface{}),
	}

	return i
}

func (i *ImageList) Name() string {
	return i.name
}

func (i *ImageList) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
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

	i.filter = ReadViewBuffer(v)

	if v, err := i.View(i.name); err == nil {
		i.GetImageList(v)
	}
}

func (i *ImageList) SetView(g *gocui.Gui) error {
	// set header panel
	if v, err := g.SetView(ImageListHeaderPanel, i.x, i.y, i.w, i.h); err != nil {
		if err != gocui.ErrUnknownView {
			panic(err)
		}

		v.Wrap = true
		v.Frame = true
		v.Title = v.Name()
		v.FgColor = gocui.AttrBold | gocui.ColorWhite
		common.OutputFormatedHeader(v, &Image{})
	}

	// set scroll panel
	v, err := g.SetView(i.name, i.x, i.y+1, i.w, i.h)
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

		i.GetImageList(v)
	}

	i.SetKeyBinding()

	//  monitoring container status interval 5s
	go func() {
		for {
			i.Update(func(g *gocui.Gui) error {
				i.Refresh(g, v)
				return nil
			})
			time.Sleep(5 * time.Second)
		}
	}()

	return nil
}

func (i *ImageList) Refresh(g *gocui.Gui, v *gocui.View) error {
	i.Update(func(g *gocui.Gui) error {
		v, err := i.View(i.name)
		if err != nil {
			panic(err)
		}
		i.GetImageList(v)
		return nil
	})

	return nil
}

func (i *ImageList) SetKeyBinding() {
	i.SetKeyBindingToPanel(i.name)

	if err := i.SetKeybinding(i.name, gocui.KeyEnter, gocui.ModNone, i.DetailImage); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, 'o', gocui.ModNone, i.DetailImage); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, 'c', gocui.ModNone, i.CreateContainerPanel); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, 'p', gocui.ModNone, i.PullImagePanel); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, 'd', gocui.ModNone, i.RemoveImage); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, gocui.KeyCtrlD, gocui.ModNone, i.RemoveDanglingImages); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, 's', gocui.ModNone, i.SaveImagePanel); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, 'i', gocui.ModNone, i.ImportImagePanel); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, gocui.KeyCtrlL, gocui.ModNone, i.LoadImagePanel); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, gocui.KeyCtrlF, gocui.ModNone, i.SearchImagePanel); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, gocui.KeyCtrlR, gocui.ModNone, i.Refresh); err != nil {
		panic(err)
	}
	if err := i.SetKeybinding(i.name, 'f', gocui.ModNone, i.Filter); err != nil {
		panic(err)
	}
}

func (i *ImageList) selected() (*Image, error) {
	v, _ := i.View(i.name)
	_, cy := v.Cursor()
	_, oy := v.Origin()

	index := oy + cy
	length := len(i.Images)

	if index >= length {
		return nil, common.NoImage
	}

	return i.Images[index], nil
}

func (i *ImageList) CreateContainerPanel(g *gocui.Gui, v *gocui.View) error {
	// get image name
	name, err := i.GetImageName()
	if err != nil {
		i.ErrMessage(err.Error(), i.name)
		return nil
	}

	// get position
	maxX, maxY := i.Size()
	x := maxX / 6
	y := maxY / 4
	w := x * 4

	labelw := 11
	fieldw := w - labelw

	// new form
	form := NewForm(g, CreateContainerPanel, x, y, w, 0)
	i.form = form

	// add func do after close
	form.AddCloseFunc(func() error {
		i.SwitchPanel(i.name)
		return nil
	})

	// add fields
	form.AddInput("Name", labelw, fieldw)
	form.AddInput("HostPort", labelw, fieldw).
		AddValidate("no specified HostPort", func(value string) bool {
			port := form.GetFieldText("Port")
			if value == "" && port != "" {
				return false
			}
			return true
		})

	form.AddInput("Port", labelw, fieldw).
		AddValidate("no specified Port", func(value string) bool {
			hostPort := form.GetFieldText("HostPort")
			if value == "" && hostPort != "" {
				return false
			}
			return true
		})

	form.AddSelectOption("VolumeType", labelw, fieldw).
		AddOptions([]string{VolumeTypeBind, VolumeTypeVolume}...)

	form.AddInput("HostVolume", labelw, fieldw).
		AddValidate("no specified HostVolume", func(value string) bool {
			volume := form.GetFieldText("Volume")
			if value == "" && volume != "" {
				return false
			}
			return true
		})
	form.AddInput("Volume", labelw, fieldw).
		AddValidate("no specified Volume", func(value string) bool {
			hostVolume := form.GetFieldText("HostVolume")
			if hostVolume != "" && value == "" {
				return false
			}
			return true
		})

	form.AddInput("Image", labelw, fieldw).
		SetText(name).
		AddValidate("no specified Image", func(value string) bool {
			return value != ""
		})

	form.AddInput("User", labelw, fieldw)
	form.AddCheckBox("Attach", labelw)
	form.AddInput("Env", labelw, fieldw)
	form.AddInput("Cmd", labelw, fieldw)
	form.AddButton("Create", i.CreateContainer)
	form.AddButton("Cancel", form.Close)

	// draw form
	form.Draw()
	return nil
}

func (i *ImageList) CreateContainer(g *gocui.Gui, v *gocui.View) error {
	if !i.form.Validate() {
		return nil
	}

	data := i.form.GetFieldTexts()
	data["VolumeType"] = i.form.GetSelectedOpt("VolumeType")

	options, err := i.Docker.NewContainerOptions(data, i.form.GetCheckBoxState("Attach"))

	if err != nil {
		i.form.Close(g, v)
		i.ErrMessage(err.Error(), i.name)
		return nil
	}

	g.Update(func(g *gocui.Gui) error {
		i.form.Close(g, v)
		i.StateMessage("container creating...")

		g.Update(func(g *gocui.Gui) error {
			defer i.CloseStateMessage()

			if err := i.Docker.CreateContainerWithOptions(options); err != nil {
				i.ErrMessage(err.Error(), i.name)
				return nil
			}

			i.Panels[ContainerListPanel].Refresh(g, v)
			i.SwitchPanel(i.name)

			return nil
		})

		return nil
	})

	return nil
}

func (i *ImageList) PullImagePanel(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := i.Size()
	x := maxX / 8
	y := maxY / 3
	w := x * 6

	labelw := 6
	fieldw := w - labelw

	// new form
	form := NewForm(g, PullImagePanel, x, y, w, 0)
	i.form = form

	// add func do after close
	form.AddCloseFunc(func() error {
		i.SwitchPanel(i.name)
		return nil
	})

	// add fields
	form.AddInput("Image", labelw, fieldw).
		AddValidate(Require.Message+"Image", Require.Validate)
	form.AddButton("Pull", i.PullImage)
	form.AddButton("Cancel", form.Close)

	// draw form
	form.Draw()
	return nil
}

func (i *ImageList) PullImage(g *gocui.Gui, v *gocui.View) error {
	if !i.form.Validate() {
		return nil
	}
	item := strings.SplitN(i.form.GetFieldTexts()["Image"], ":", 2)

	name := item[0]
	var tag string

	if len(item) == 1 {
		tag = "latest"
	} else {
		tag = item[1]
	}

	i.form.Close(g, v)

	f := func() error {
		options := docker.PullImageOptions{
			Repository: name,
			Tag:        tag,
		}

		if err := i.Docker.PullImageWithOptions(options); err != nil {
			return err
		}

		i.Refresh(g, v)

		return nil
	}

	i.AddTask(PullImage.String(), f)

	return nil
}

func (i *ImageList) DetailImage(g *gocui.Gui, v *gocui.View) error {
	image, err := i.selected()
	if err != nil {
		i.ErrMessage(err.Error(), i.name)
		return nil
	}

	img, err := i.Docker.InspectImage(image.ID)
	if err != nil {
		i.ErrMessage(err.Error(), i.name)
		return nil
	}

	i.PopupDetailPanel(g, v)

	v, err = g.View(DetailPanel)
	if err != nil {
		panic(err)
	}

	v.Clear()
	v.SetOrigin(0, 0)
	v.SetCursor(0, 0)
	fmt.Fprint(v, common.StructToJson(img))

	return nil
}

func (i *ImageList) SaveImagePanel(g *gocui.Gui, v *gocui.View) error {
	name, err := i.GetImageName()
	if err != nil {
		i.ErrMessage(err.Error(), i.name)
		return nil
	}

	maxX, maxY := i.Size()
	x := maxX / 8
	y := maxY / 3
	w := x * 6

	labelw := 6
	fieldw := w - labelw

	// new form
	form := NewForm(g, SaveImagePanel, x, y, w, 0)
	i.form = form

	// add func do after close
	form.AddCloseFunc(func() error {
		i.SwitchPanel(i.name)
		return nil
	})

	// add fields
	form.AddInput("Path", labelw, fieldw).
		AddValidate(Require.Message+"Path", Require.Validate)
	form.AddInput("Image", labelw, fieldw).
		AddValidate(Require.Message+"Image", Require.Validate).
		SetText(name)
	form.AddButton("Save", i.SaveImage)
	form.AddButton("Cancel", form.Close)

	// draw form
	form.Draw()

	return nil
}

func (i *ImageList) SaveImage(g *gocui.Gui, v *gocui.View) error {
	if !i.form.Validate() {
		return nil
	}
	data := i.form.GetFieldTexts()

	g.Update(func(g *gocui.Gui) error {
		i.form.Close(g, v)
		i.StateMessage("image saving....")

		g.Update(func(g *gocui.Gui) error {
			defer i.CloseStateMessage()

			file, err := os.OpenFile(data["Path"], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
			if err != nil {
				i.ErrMessage(err.Error(), i.name)
				return nil
			}
			defer file.Close()

			options := docker.ExportImageOptions{
				Name:         data["Image"],
				OutputStream: file,
			}

			if err := i.Docker.SaveImageWithOptions(options); err != nil {
				i.ErrMessage(err.Error(), i.name)
				return nil
			}

			i.SwitchPanel(i.name)

			return nil
		})

		return nil
	})

	return nil
}

func (i *ImageList) ImportImagePanel(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := i.Size()
	x := maxX / 8
	y := maxY / 3
	w := x * 6

	labelw := 11
	fieldw := w - labelw

	// new form
	form := NewForm(g, ImportImagePanel, x, y, w, 0)
	i.form = form

	// add func do after close
	form.AddCloseFunc(func() error {
		i.SwitchPanel(i.name)
		return nil
	})

	// add fields
	form.AddInput("Repository", labelw, fieldw).
		AddValidate(Require.Message+"Repository", Require.Validate)
	form.AddInput("Path", labelw, fieldw).
		AddValidate(Require.Message+"Path", Require.Validate)
	form.AddInput("Tag", labelw, fieldw)
	form.AddButton("Import", i.ImportImage)
	form.AddButton("Cancel", form.Close)

	// draw form
	form.Draw()

	return nil
}

func (i *ImageList) ImportImage(g *gocui.Gui, v *gocui.View) error {
	if !i.form.Validate() {
		return nil
	}

	data := i.form.GetFieldTexts()
	options := docker.ImportImageOptions{
		Repository: data["Repository"],
		Source:     data["Path"],
		Tag:        data["Tag"],
	}

	g.Update(func(g *gocui.Gui) error {
		i.form.Close(g, v)
		i.StateMessage("image importing....")

		g.Update(func(g *gocui.Gui) error {
			defer i.CloseStateMessage()

			if err := i.Docker.ImportImageWithOptions(options); err != nil {
				i.ErrMessage(err.Error(), i.name)
				return nil
			}

			i.Refresh(g, v)
			i.SwitchPanel(i.name)

			return nil
		})

		return nil
	})

	return nil
}

func (i *ImageList) LoadImagePanel(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := i.Size()
	x := maxX / 8
	y := maxY / 3
	w := x * 6

	labelw := 6
	fieldw := w - labelw

	// new form
	form := NewForm(g, LoadImagePanel, x, y, w, 0)
	i.form = form

	// add func do after close
	form.AddCloseFunc(func() error {
		i.SwitchPanel(i.name)
		return nil
	})

	// add fields
	form.AddInput("Path", labelw, fieldw).
		AddValidate(Require.Message+"Path", Require.Validate)
	form.AddButton("Load", i.LoadImage)
	form.AddButton("Cancel", form.Close)

	// draw form
	form.Draw()
	return nil
}

func (i *ImageList) LoadImage(g *gocui.Gui, v *gocui.View) error {
	if !i.form.Validate() {
		return nil
	}

	path := i.form.GetFieldTexts()["Path"]

	g.Update(func(g *gocui.Gui) error {
		i.form.Close(g, v)
		i.StateMessage("image loading....")

		g.Update(func(g *gocui.Gui) error {

			defer i.CloseStateMessage()
			if err := i.Docker.LoadImageWithPath(path); err != nil {
				i.ErrMessage(err.Error(), i.name)
				return nil
			}

			i.Refresh(g, v)
			i.SwitchPanel(i.name)

			return nil
		})

		return nil
	})

	return nil
}

func (i *ImageList) SearchImagePanel(g *gocui.Gui, v *gocui.View) error {
	i.name = g.CurrentView().Name()

	maxX, maxY := g.Size()
	x := maxX / 8
	y := maxY / 4
	w := maxX - x
	h := y + 2

	NewSearchImage(i.Gui, SearchImagePanel, Position{x, y, w, h})
	return nil
}

func (i *ImageList) GetImageList(v *gocui.View) {
	v.Clear()
	i.Images = make([]*Image, 0)

	for _, image := range i.Docker.Images(docker.ListImagesOptions{}) {
		for _, repoTag := range image.RepoTags {
			repo, tag := common.ParseRepoTag(repoTag)

			if i.filter != "" {
				name := fmt.Sprintf("%s:%s", repo, tag)
				if strings.Index(strings.ToLower(name), strings.ToLower(i.filter)) == -1 {
					continue
				}
			}

			id := image.ID[7:19]
			created := common.ParseDateToString(image.Created)
			size := common.ParseSizeToString(image.Size)

			image := &Image{
				ID:      id,
				Repo:    repo,
				Tag:     tag,
				Created: created,
				Size:    size,
			}

			i.Images = append(i.Images, image)

			common.OutputFormatedLine(v, image)
		}
	}
}

func (i *ImageList) GetImageName() (string, error) {
	image, err := i.selected()
	if err != nil {
		return "", err
	}

	var name string
	if image.Repo == "<none>" || image.Tag == "<none>" {
		name = image.ID
	} else {
		name = fmt.Sprintf("%s:%s", image.Repo, image.Tag)
	}

	return name, nil
}

func (i *ImageList) RemoveImage(g *gocui.Gui, v *gocui.View) error {
	name, err := i.GetImageName()
	if err != nil {
		i.ErrMessage(err.Error(), i.name)
		return nil
	}

	i.ConfirmMessage("Are you sure you want to remove this image?", i.name, func() error {
		defer i.Refresh(g, v)
		if err := i.Docker.RemoveImageWithName(name); err != nil {
			i.ErrMessage(err.Error(), i.name)
			return nil
		}

		return nil
	})

	return nil
}

func (i *ImageList) RemoveDanglingImages(g *gocui.Gui, v *gocui.View) error {
	if len(i.Images) == 0 {
		i.ErrMessage(common.NoImage.Error(), i.name)
		return nil
	}

	i.ConfirmMessage("Are you sure you want to remove dangling images?", i.name, func() error {
		defer i.Refresh(g, v)
		if err := i.Docker.RemoveDanglingImages(); err != nil {
			i.ErrMessage(err.Error(), i.name)
			return nil
		}

		return nil
	})

	return nil
}

func (i *ImageList) Filter(g *gocui.Gui, lv *gocui.View) error {
	isReset := false
	closePanel := func(g *gocui.Gui, v *gocui.View) error {
		if isReset {
			i.filter = ""
		} else {
			lv.SetCursor(0, 0)
			i.filter = ReadViewBuffer(v)
		}
		if v, err := i.View(i.name); err == nil {
			i.GetImageList(v)
		}

		if err := g.DeleteView(v.Name()); err != nil {
			panic(err)
		}

		g.DeleteKeybindings(v.Name())
		i.SwitchPanel(i.name)
		return nil
	}

	reset := func(g *gocui.Gui, v *gocui.View) error {
		isReset = true
		return closePanel(g, v)
	}

	if err := i.NewFilterPanel(i, reset, closePanel); err != nil {
		panic(err)
	}

	return nil
}
