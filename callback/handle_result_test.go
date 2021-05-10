package callback

import "testing"

func Test_Handled_follows_or_logic_table(t *testing.T) {
	result := Handled

	if result.Or(Handled) != Handled {
		t.Fatalf("handled or handled should be handled")
	}

	if result.Or(HandledAndStop) != HandledAndStop {
		t.Fatalf("handled or HandledAndStop should be HandledAndStop")
	}

	if result.Or(NotHandled) != Handled {
		t.Fatalf("handled or NotHandled should be handled")
	}

	if result.Or(NotHandledAndStop) != HandledAndStop {
		t.Fatalf("handled or NotHandledAndStop should be HandledAndStop")
	}
}

func Test_HandledAndStop_follows_or_logic_table(t *testing.T) {
	result := HandledAndStop

	if result.Or(Handled) != HandledAndStop {
		t.Fatalf("HandledAndStop or handled should be HandledAndStop")
	}

	if result.Or(HandledAndStop) != HandledAndStop {
		t.Fatalf("HandledAndStop or HandledAndStop should be HandledAndStop")
	}

	if result.Or(NotHandled) != HandledAndStop {
		t.Fatalf("HandledAndStop or NotHandled should be HandledAndStop")
	}

	if result.Or(NotHandledAndStop) != HandledAndStop {
		t.Fatalf("HandledAndStop or NotHandledAndStop should be HandledAndStop")
	}
}

func Test_NotHandled_follows_or_logic_table(t *testing.T) {
	result := NotHandled

	if result.Or(Handled) != Handled {
		t.Fatalf("NotHandled or handled should be handled")
	}

	if result.Or(HandledAndStop) != HandledAndStop {
		t.Fatalf("NotHandled or HandledAndStop should be HandledAndStop")
	}

	if result.Or(NotHandled) != NotHandled {
		t.Fatalf("NotHandled or NotHandled should be NotHandled")
	}

	if result.Or(NotHandledAndStop) != NotHandledAndStop {
		t.Fatalf("NotHandled or NotHandledAndStop should be NotHandledAndStop")
	}
}

func Test_NotHandledAndStop_follows_or_logic_table(t *testing.T) {
	result := NotHandledAndStop

	if result.Or(Handled) != HandledAndStop {
		t.Fatalf("NotHandledAndStop or handled should be HandledAndStop")
	}

	if result.Or(HandledAndStop) != HandledAndStop {
		t.Fatalf("NotHandledAndStop or HandledAndStop should be HandledAndStop")
	}

	if result.Or(NotHandled) != NotHandledAndStop {
		t.Fatalf("NotHandledAndStop or NotHandled should be NotHandledAndStop")
	}

	if result.Or(NotHandledAndStop) != NotHandledAndStop {
		t.Fatalf("NotHandledAndStop or NotHandledAndStop should be NotHandledAndStop")
	}
}

func Test_Handled_follows_and_logic_table(t *testing.T) {
	result := Handled

	if result.And(Handled) != Handled {
		t.Fatalf("handled and handled should be handled")
	}

	if result.And(HandledAndStop) != HandledAndStop {
		t.Fatalf("handled and HandledAndStop should be HandledAndStop")
	}

	if result.And(NotHandled) != NotHandled {
		t.Fatalf("handled and NotHandled should be NotHandled")
	}

	if result.And(NotHandledAndStop) != NotHandledAndStop {
		t.Fatalf("handled and NotHandledAndStop should be NotHandledAndStop")
	}
}

func Test_HandledAndStop_follows_and_logic_table(t *testing.T) {
	result := HandledAndStop

	if result.And(Handled) != HandledAndStop {
		t.Fatalf("HandledAndStop and handled should be HandledAndStop")
	}

	if result.And(HandledAndStop) != HandledAndStop {
		t.Fatalf("HandledAndStop and HandledAndStop should be HandledAndStop")
	}

	if result.And(NotHandled) != NotHandledAndStop {
		t.Fatalf("HandledAndStop and NotHandled should be NotHandledAndStop")
	}

	if result.And(NotHandledAndStop) != NotHandledAndStop {
		t.Fatalf("HandledAndStop and NotHandledAndStop should be NotHandledAndStop")
	}
}

func Test_NotHandled_follows_and_logic_table(t *testing.T) {
	result := NotHandled

	if result.And(Handled) != NotHandled {
		t.Fatalf("NotHandled and handled should be NotHandled")
	}

	if result.And(HandledAndStop) != NotHandledAndStop {
		t.Fatalf("NotHandled and HandledAndStop should be NotHandledAndStop")
	}

	if result.And(NotHandled) != NotHandled {
		t.Fatalf("NotHandled and NotHandled should be NotHandled")
	}

	if result.And(NotHandledAndStop) != NotHandledAndStop {
		t.Fatalf("NotHandled and NotHandledAndStop should be NotHandledAndStop")
	}
}

func Test_NotHandledAndStop_follows_and_logic_table(t *testing.T) {
	result := NotHandledAndStop

	if result.And(Handled) != NotHandledAndStop {
		t.Fatalf("NotHandledAndStop and handled should be NotHandledAndStop")
	}

	if result.And(HandledAndStop) != NotHandledAndStop {
		t.Fatalf("NotHandledAndStop and HandledAndStop should be NotHandledAndStop")
	}

	if result.And(NotHandled) != NotHandledAndStop {
		t.Fatalf("NotHandledAndStop and NotHandled should be NotHandledAndStop")
	}

	if result.And(NotHandledAndStop) != NotHandledAndStop {
		t.Fatalf("NotHandledAndStop and NotHandledAndStop should be NotHandledAndStop")
	}
}