package callback

import "testing"

func Test_Handled_follows_or_logic_table(t *testing.T) {
	result := Handled

	if result.Or(Handled) != Handled {
		t.Fatalf("Handled or Handled should be Handled")
	}

	if result.Or(HandledAndStop) != HandledAndStop {
		t.Fatalf("Handled or HandledAndStop should be HandledAndStop")
	}

	if result.Or(NotHandled) != Handled {
		t.Fatalf("Handled or NotHandled should be Handled")
	}

	if result.Or(NotHandledAndStop) != HandledAndStop {
		t.Fatalf("Handled or NotHandledAndStop should be HandledAndStop")
	}
}

func Test_HandledAndStop_follows_or_logic_table(t *testing.T) {
	result := HandledAndStop

	if result.Or(Handled) != HandledAndStop {
		t.Fatalf("HandledAndStop or Handled should be HandledAndStop")
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
		t.Fatalf("NotHandled or Handled should be Handled")
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
		t.Fatalf("NotHandledAndStop or Handled should be HandledAndStop")
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
		t.Fatalf("Handled and Handled should be Handled")
	}

	if result.And(HandledAndStop) != HandledAndStop {
		t.Fatalf("Handled and HandledAndStop should be HandledAndStop")
	}

	if result.And(NotHandled) != NotHandled {
		t.Fatalf("Handled and NotHandled should be NotHandled")
	}

	if result.And(NotHandledAndStop) != NotHandledAndStop {
		t.Fatalf("Handled and NotHandledAndStop should be NotHandledAndStop")
	}
}

func Test_HandledAndStop_follows_and_logic_table(t *testing.T) {
	result := HandledAndStop

	if result.And(Handled) != HandledAndStop {
		t.Fatalf("HandledAndStop and Handled should be HandledAndStop")
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
		t.Fatalf("NotHandled and Handled should be NotHandled")
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
		t.Fatalf("NotHandledAndStop and Handled should be NotHandledAndStop")
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