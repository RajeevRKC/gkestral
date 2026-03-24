package gemini

import "testing"

func TestClassifyTask(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		hasTools  bool
		hasImages bool
		want      TaskClass
	}{
		{
			name:      "image request",
			prompt:    "Draw a cat",
			hasImages: true,
			want:      TaskImageGeneration,
		},
		{
			name:   "code generation",
			prompt: "Write a function to sort an array",
			want:   TaskCodeGeneration,
		},
		{
			name:   "code debugging",
			prompt: "I have a bug in my program where the variable is undefined",
			want:   TaskCodeGeneration,
		},
		{
			name:   "research factual question",
			prompt: "What is the population of Riyadh?",
			want:   TaskResearch,
		},
		{
			name:   "research history",
			prompt: "Tell me about the history of mangroves in the Arabian Gulf",
			want:   TaskResearch,
		},
		{
			name:   "fast extraction short prompt",
			prompt: "Extract the email addresses from this text",
			want:   TaskFastExtraction,
		},
		{
			name:   "summarization short prompt",
			prompt: "Summarize the following paragraph",
			want:   TaskFastExtraction,
		},
		{
			name:   "deep reasoning with analysis keywords",
			prompt: "Analyze the trade-off between caching strategies and their implications for system architecture",
			want:   TaskDeepReasoning,
		},
		{
			name:   "conversation fallback",
			prompt: "Hello, how are you today?",
			want:   TaskConversation,
		},
		{
			name:   "large context",
			prompt: string(make([]byte, 60000)), // >50K chars
			want:   TaskLargeContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyTask(tt.prompt, tt.hasTools, tt.hasImages)
			if got != tt.want {
				t.Errorf("ClassifyTask() = %v (%s), want %v (%s)", got, got.String(), tt.want, tt.want.String())
			}
		})
	}
}

func TestClassifyTask_DeepReasoningLongPrompt(t *testing.T) {
	// Long prompt (>2000 chars) with reasoning keywords should classify as deep reasoning.
	longPrompt := "Please analyze the following architecture in depth and evaluate the trade-offs: " + string(make([]byte, 2500))
	got := ClassifyTask(longPrompt, false, false)
	if got != TaskDeepReasoning {
		t.Errorf("long analysis prompt = %v, want DeepReasoning", got)
	}
}

func TestRouter_RouteModel(t *testing.T) {
	router := NewRouter()

	tests := []struct {
		name      string
		taskClass TaskClass
		wantModel string
		wantThink bool
		wantGnd   bool
	}{
		{
			name:      "deep reasoning routes to Pro",
			taskClass: TaskDeepReasoning,
			wantModel: "gemini-3.1-pro-preview",
			wantThink: true,
		},
		{
			name:      "fast extraction routes to Flash",
			taskClass: TaskFastExtraction,
			wantModel: "gemini-3.1-flash",
		},
		{
			name:      "image generation routes to image preview",
			taskClass: TaskImageGeneration,
			wantModel: "gemini-3.1-flash-image-preview",
		},
		{
			name:      "code generation routes to Flash with thinking",
			taskClass: TaskCodeGeneration,
			wantModel: "gemini-3.1-flash",
			wantThink: true,
		},
		{
			name:      "research routes to Pro with grounding",
			taskClass: TaskResearch,
			wantModel: "gemini-3.1-pro-preview",
			wantGnd:   true,
		},
		{
			name:      "conversation routes to cheapest Flash",
			taskClass: TaskConversation,
			wantModel: "gemini-2.5-flash",
		},
		{
			name:      "large context routes to 2.5 Pro",
			taskClass: TaskLargeContext,
			wantModel: "gemini-2.5-pro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, rule, err := router.RouteModel(tt.taskClass)
			if err != nil {
				t.Fatalf("RouteModel error: %v", err)
			}
			if model.ID != tt.wantModel {
				t.Errorf("model = %q, want %q", model.ID, tt.wantModel)
			}
			if rule.RequiresThinking != tt.wantThink {
				t.Errorf("thinking = %v, want %v", rule.RequiresThinking, tt.wantThink)
			}
			if rule.RequiresGrounding != tt.wantGnd {
				t.Errorf("grounding = %v, want %v", rule.RequiresGrounding, tt.wantGnd)
			}
		})
	}
}

func TestRouter_CustomRouting(t *testing.T) {
	// Override conversation to use Pro instead of Flash.
	router := NewCustomRouter([]RoutingRule{
		{
			TaskClass:      TaskConversation,
			PreferredModel: "gemini-2.5-pro",
		},
	})

	model, _, err := router.RouteModel(TaskConversation)
	if err != nil {
		t.Fatalf("RouteModel error: %v", err)
	}
	if model.ID != "gemini-2.5-pro" {
		t.Errorf("model = %q, want %q", model.ID, "gemini-2.5-pro")
	}
}

func TestRouter_SetRule(t *testing.T) {
	router := NewRouter()

	// Override code generation.
	router.SetRule(RoutingRule{
		TaskClass:      TaskCodeGeneration,
		PreferredModel: "gemini-2.5-pro",
	})

	model, _, err := router.RouteModel(TaskCodeGeneration)
	if err != nil {
		t.Fatalf("RouteModel error: %v", err)
	}
	if model.ID != "gemini-2.5-pro" {
		t.Errorf("model = %q, want %q", model.ID, "gemini-2.5-pro")
	}
}

func TestRouter_GetRule(t *testing.T) {
	router := NewRouter()

	rule := router.GetRule(TaskDeepReasoning)
	if rule == nil {
		t.Fatal("expected rule for DeepReasoning")
	}
	if rule.ThinkingLevel != ThinkingLevelHigh {
		t.Errorf("thinking level = %q, want %q", rule.ThinkingLevel, ThinkingLevelHigh)
	}

	// Unknown task class should return nil.
	rule = router.GetRule(TaskClass(99))
	if rule != nil {
		t.Error("expected nil for unknown task class")
	}
}

func TestTaskClass_String(t *testing.T) {
	tests := []struct {
		tc   TaskClass
		want string
	}{
		{TaskDeepReasoning, "deep_reasoning"},
		{TaskFastExtraction, "fast_extraction"},
		{TaskImageGeneration, "image_generation"},
		{TaskCodeGeneration, "code_generation"},
		{TaskResearch, "research"},
		{TaskConversation, "conversation"},
		{TaskLargeContext, "large_context"},
		{TaskClass(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.tc.String(); got != tt.want {
				t.Errorf("TaskClass(%d).String() = %q, want %q", int(tt.tc), got, tt.want)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s      string
		subs   []string
		want   bool
	}{
		{"hello world", []string{"world"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"analyze this code", []string{"analyze", "evaluate"}, true},
		{"", []string{"anything"}, false},
		{"something", []string{}, false},
	}

	for _, tt := range tests {
		got := containsAny(tt.s, tt.subs)
		if got != tt.want {
			t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.subs, got, tt.want)
		}
	}
}

func TestDefaultRoutingRules_Coverage(t *testing.T) {
	rules := defaultRoutingRules()

	// Every task class should have a rule.
	allClasses := []TaskClass{
		TaskDeepReasoning, TaskFastExtraction, TaskImageGeneration,
		TaskCodeGeneration, TaskResearch, TaskConversation, TaskLargeContext,
	}

	for _, tc := range allClasses {
		if _, ok := rules[tc]; !ok {
			t.Errorf("missing default rule for %s", tc.String())
		}
	}
}
