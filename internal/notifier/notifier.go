package notifier

type Notifier struct {
	PfdChangeNotifier *PfdChangeNotifier
}

func NewNotifier() *Notifier {
	n := &Notifier{}
	if n.PfdChangeNotifier = NewPfdChangeNotifier(); n.PfdChangeNotifier == nil {
		return nil
	}
	return n
}
