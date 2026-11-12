package notifier

type Notifier struct {
	PfdChangeNotifier *PfdChangeNotifier
}

func NewNotifier(verify bool) (*Notifier, error) {
	var err error
	n := &Notifier{}
	if n.PfdChangeNotifier, err = NewPfdChangeNotifier(verify); err != nil {
		return nil, err
	}
	return n, nil
}
