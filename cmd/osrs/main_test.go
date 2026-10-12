package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLI(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"xp", "--level", "70", "--target", "99", "--json"}, &out); err != nil {
		t.Fatal(err)
	}
	var got xpOut
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.XP != 737_627 || got.Remaining != 12_296_804 {
		t.Errorf("xp = %+v", got)
	}

	out.Reset()
	if err := run(context.Background(), []string{"combat", "--attack", "60", "--strength", "99", "--hitpoints", "90", "--prayer", "52"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "combat level 80\n" {
		t.Errorf("combat text = %q", out.String())
	}
}

func TestReorder(t *testing.T) {
	var in playerIn
	if _, err := parseFlags("hiscores", "", &in, []string{"Lynx", "--mode", "ironman", "Titan"}); err != nil {
		t.Fatal(err)
	}
	if in.Player != "Lynx Titan" || in.Mode != "ironman" {
		t.Errorf("parsed %+v", in)
	}
}

func TestMCP(t *testing.T) {
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	cmds := commands()
	t.Setenv("OSRS_NO_CACHE", "1")
	d, err := newDeps()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newMCPServer(d, cmds).Connect(ctx, serverT, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != len(cmds) {
		t.Errorf("got %d tools, want %d", len(tools.Tools), len(cmds))
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "xp", Arguments: map[string]any{"level": 70, "target": 99}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(res.StructuredContent)
	if res.IsError || !strings.Contains(string(b), `"remaining":12296804`) {
		t.Errorf("xp tool = %s (error %v)", b, res.IsError)
	}

	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "combat", Arguments: map[string]any{"attack": 200}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("combat with attack 200: want tool error")
	}
}
