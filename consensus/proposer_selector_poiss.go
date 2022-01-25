package consensus

import (
	"math/big"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

const precision = 256

var (
	blocksPerEpoch = big.NewInt(5)
	expNumCoef     []*big.Int
	expDenoCoef    []*big.Int
)

func init() {
	parse := func(coefs []string) []*big.Int {
		out := make([]*big.Int, len(coefs))
		for i, coef := range coefs {
			c, ok := new(big.Int).SetString(coef, 10)
			if !ok {
				panic("could not parse exp paramemter")
			}
			c = c.Lsh(c, precision-128)
			out[i] = c
		}
		return out
	}

	num := []string{
		"-648770010757830093818553637600",
		"67469480939593786226847644286976",
		"-3197587544499098424029388939001856",
		"89244641121992890118377641805348864",
		"-1579656163641440567800982336819953664",
		"17685496037279256458459817590917169152",
		"-115682590513835356866803355398940131328",
		"340282366920938463463374607431768211456",
	}
	expNumCoef = parse(num)

	deno := []string{
		"1225524182432722209606361",
		"114095592300906098243859450",
		"5665570424063336070530214243",
		"194450132448609991765137938448",
		"5068267641632683791026134915072",
		"104716890604972796896895427629056",
		"1748338658439454459487681798864896",
		"23704654329841312470660182937960448",
		"259380097567996910282699886670381056",
		"2250336698853390384720606936038375424",
		"14978272436876548034486263159246028800",
		"72144088983913131323343765784380833792",
		"224599776407103106596571252037123047424",
		"340282366920938463463374607431768211456",
	}
	expDenoCoef = parse(deno)
}

func expneg(x *big.Int) *big.Int {
	num := polyval(expNumCoef, x)
	deno := polyval(expDenoCoef, x)

	num = num.Lsh(num, precision)
	return num.Div(num, deno)
}

func polyval(p []*big.Int, x *big.Int) *big.Int {
	res := new(big.Int).Set(p[0])
	tmp := new(big.Int)
	for _, c := range p[1:] {
		tmp = tmp.Mul(res, x)
		res = res.Rsh(tmp, precision)
		res = res.Add(res, c)
	}

	return res
}

func lambda(weight, totalWeight *big.Int) *big.Int {
	lam := new(big.Int).Mul(weight, blocksPerEpoch)
	lam = lam.Lsh(lam, precision)
	lam = lam.Div(lam, totalWeight)
	return lam
}

var MaxWinCount = 3 * int64(5)

type poiss struct {
	lam  *big.Int
	pmf  *big.Int
	icdf *big.Int

	tmp *big.Int

	k uint64
}

func newPoiss(lambda *big.Int) (*poiss, *big.Int) {
	elam := expneg(lambda) // Q.256
	pmf := new(big.Int).Set(elam)

	icdf := big.NewInt(1)
	icdf = icdf.Lsh(icdf, precision)
	icdf = icdf.Sub(icdf, pmf)

	k := uint64(0)

	p := &poiss{
		lam: lambda,
		pmf: pmf,

		tmp:  elam,
		icdf: icdf,

		k: k,
	}

	return p, icdf
}

func (p *poiss) next() *big.Int {
	p.k++
	p.tmp.SetUint64(p.k)

	p.pmf = p.pmf.Div(p.pmf, p.tmp)
	p.tmp = p.tmp.Mul(p.pmf, p.lam)
	p.pmf = p.pmf.Rsh(p.tmp, precision)

	p.icdf = p.icdf.Sub(p.icdf, p.pmf)
	return p.icdf
}

type proposerSelectorPoiss struct {
	crypt  tpcrt.CryptService
	hasher tpcmm.Hasher
}

func newProposerSelectorPoiss(crypt tpcrt.CryptService, hasher tpcmm.Hasher) *proposerSelectorPoiss {
	return &proposerSelectorPoiss{crypt: crypt, hasher: hasher}
}

func (ep *proposerSelectorPoiss) ComputeVRF(priKey tpcrtypes.PrivateKey, data []byte) ([]byte, error) {
	return ep.crypt.Sign(priKey, data)
}

func (ep *proposerSelectorPoiss) SelectProposer(VRFProof []byte, weight *big.Int, totalWeight *big.Int) int64 {
	h := ep.hasher.Compute(string(VRFProof))

	lhs := new(big.Int).SetBytes(h[:])

	lam := lambda(weight, totalWeight)

	p, rhs := newPoiss(lam)

	var j int64
	for lhs.Cmp(rhs) < 0 && j < MaxWinCount {
		rhs = p.next()
		j++
	}

	return j
}
