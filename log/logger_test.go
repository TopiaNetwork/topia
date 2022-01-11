package log

import (
	"testing"

	"github.com/stretchr/testify/assert"

	logcomm "github.com/TopiaNetwork/topia/log/common"
)

func TestCreateMainLogger(t *testing.T) {
	i := 100
	str := "TestCreate"
	log, err := CreateMainLogger(logcomm.DebugLevel, JSONFormat, StdErrOutput, "")
	assert.Equal(t, err, nil)
	log.Debug("TestCreateMainLogger ok")
	log.Info("TestCreateMainLogger ok")
	log.Infof("TestCreateMainLogger ok i=%d, str=%s", i, str)

	log.UpdateLoggerLevel(logcomm.InfoLevel)

	log.Debug("TestCreateMainLogger ok after update")
	log.Info("TestCreateMainLogger ok after update")
}

func TestCreateModuleLogger(t *testing.T) {
	log, err := CreateMainLogger(logcomm.DebugLevel, JSONFormat, StdErrOutput, "")
	assert.Equal(t, err, nil)
	log.Debug("MainLogger ok")
	log.Info("MainLogger ok")

	ml := CreateModuleLogger(logcomm.InfoLevel, "P2P", log)

	ml.Debug("P2P Logger ok after update")
	ml.Info("P2P Logger ok after update")
}
