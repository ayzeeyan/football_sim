package brain

// Base is the pre-trained starting brain, fitted offline by cmd/pretrain
// from simulated seasons and committed as the shipped artifact. Runtime
// post-training continues from here: the brain never stops learning, but
// every career starts from the same good base instead of from zero.
//
// Fitted from 8 seasons of self-play (18024 training rows).
func Base() *Model {
	return &Model{
		Weights: [NumFeatures]float64{
			0.043823,
			0.611125,
			1.589767,
			0.049250,
			-0.040700,
			0.147576,
			-0.129562,
			0.600993,
		},
		Bias:    0.359172,
		Samples: 18024,
		LossEMA: 2.297970,
	}
}
