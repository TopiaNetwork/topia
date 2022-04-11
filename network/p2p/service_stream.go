package p2p

import (
	"bufio"
	"context"
	"fmt"
	"io"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/network"

	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network/message"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
)

type P2PStreamService struct {
	ctx        context.Context
	log        tplog.Logger
	p2pService *P2PService
}

func NewP2PStreamService(ctx context.Context, log tplog.Logger, p2pService *P2PService) *P2PStreamService {
	return &P2PStreamService{
		ctx:        ctx,
		log:        tplog.CreateModuleLogger(logcomm.InfoLevel, "P2PStreamService", log),
		p2pService: p2pService,
	}
}

func (ps *P2PStreamService) handleIncomingStream(stream network.Stream) {
	ps.log.Infof("Received stream ID=%s, protocol=%s localID=%s remoteID=%s",
		stream.ID(), stream.Protocol(), stream.Conn().LocalPeer().String(), stream.Conn().RemotePeer().String())

	reader := ggio.NewDelimitedReader(stream, tpnetprotoc.StreamMaxMsgSize)

	for {
		select {
		case <-ps.ctx.Done():
			return
		default:
		}

		var streamMsg message.NetworkMessage
		err := reader.ReadMsg(&streamMsg)
		if err != nil {
			ps.log.Errorf("failed to read message: %s and stream %s will be reset", err.Error(), stream.ID())
			if err == io.EOF {
				stream.Close()
				ps.log.Debugf("close the stream: %s", stream.ID())
				return
			}
			stream.Reset()
			return
		}

		if streamMsg.Size() > tpnetprotoc.StreamMaxMsgSize {
			stream.Reset()
			ps.log.Errorf("received message exceeded permissible message maxSize FromPeerID=%s, ProtocolID=%s, ModuleName=%s, size=%s, maxSize=%d",
				streamMsg.FromPeerID,
				streamMsg.ProtocolID,
				streamMsg.ModuleName,
				streamMsg.Size(),
				tpnetprotoc.StreamMaxMsgSize)
			return
		}

		ps.handleStreamMessage(stream, &streamMsg)
	}
}

func (ps *P2PStreamService) handleStreamMessage(stream network.Stream, streamMsg *message.NetworkMessage) error {
	if streamMsg.ModuleName == "" {
		err := fmt.Errorf("invalid module name for stream message %s", stream.ID())
		ps.log.Error(err.Error())
		return err
	}

	return ps.p2pService.dispatch(streamMsg.ModuleName, streamMsg.Data)
}

func (ps *P2PStreamService) handleIncomingStreamWithResp(stream network.Stream) {
	defer stream.Close() //nolint:errcheck

	ps.log.Infof("Received stream ID=%s, protocol=%s localID=%s remoteID=%s",
		stream.ID(), stream.Protocol(), stream.Conn().LocalPeer().String(), stream.Conn().RemotePeer().String())

	reader := ggio.NewDelimitedReader(stream, tpnetprotoc.StreamMaxMsgSize)
	var streamMsg message.NetworkMessage
	err := reader.ReadMsg(&streamMsg)
	if err != nil {
		ps.log.Errorf("failed to read message: %s and stream %s will be reset", err.Error(), stream.ID())
		if err == io.EOF {
			stream.Close()
			ps.log.Debugf("close the stream: %s", stream.ID())
			return
		}
		stream.Reset()
		return
	}

	if streamMsg.Size() > tpnetprotoc.StreamMaxMsgSize {
		stream.Reset()
		ps.log.Errorf("received message exceeded permissible message maxSize FromPeerID=%s, ProtocolID=%s, ModuleName=%s, size=%d, maxSize=%d",
			streamMsg.FromPeerID,
			streamMsg.ProtocolID,
			streamMsg.ModuleName,
			streamMsg.Size(),
			tpnetprotoc.StreamMaxMsgSize)
		return
	}

	err = ps.handleStreamMessageWithResp(stream, &streamMsg)
	if err != nil {
		stream.Reset()
		ps.log.Errorf("handleStreamMessageWithResp error: FromPeerID=%s, ProtocolID=%s, ModuleName=%s, %v",
			streamMsg.FromPeerID,
			streamMsg.ProtocolID,
			streamMsg.ModuleName,
			err)
	}

	return
}

func (ps *P2PStreamService) handleStreamMessageWithResp(stream network.Stream, streamMsg *message.NetworkMessage) error {
	if streamMsg.ModuleName == "" {
		err := fmt.Errorf("invalid module name for stream message %s", stream.ID())
		ps.log.Error(err.Error())
		return err
	}

	resp, err := ps.p2pService.dispatchAndWaitResp(streamMsg.ModuleName, streamMsg)
	if err != nil {
		return err
	}

	respMsg := &message.NetworkMessage{
		FromPeerID: ps.p2pService.host.ID().String(),
		ProtocolID: string(stream.Protocol()),
		ModuleName: streamMsg.ModuleName,
		Data:       resp.([]byte),
	}

	return ps.writeMessage(stream, respMsg)
}

func (ps *P2PStreamService) writeMessage(stream network.Stream, streamMsg *message.NetworkMessage) error {
	w := bufio.NewWriter(stream)
	ggw := ggio.NewDelimitedWriter(w)

	streamMsg.FromPeerID = ps.p2pService.host.ID().String()
	if err := ggw.WriteMsg(streamMsg); err != nil {
		ps.log.Errorf("write to stream %s err %s", stream.ID(), err.Error())
		return err
	}

	return w.Flush()
}

func (ps *P2PStreamService) readMessage(stream network.Stream) (*message.NetworkMessage, error) {
	r := bufio.NewReader(NewStreamReader(stream, tpnetprotoc.ReadResMinSpeed, tpnetprotoc.ReadResDeadline))
	ggr := ggio.NewDelimitedReader(r, tpnetprotoc.StreamMaxMsgSize)

	var streamMsg message.NetworkMessage
	err := ggr.ReadMsg(&streamMsg)
	if err != nil {
		err = fmt.Errorf("failed to read message: %s and stream %s will be reset", err.Error(), stream.ID())
		ps.log.Errorf(err.Error())
		if err == io.EOF {
			stream.Close()
			ps.log.Debugf("close the stream: %s", stream.ID())
			return nil, err
		}
		stream.Reset()
		return nil, err
	}

	if streamMsg.Size() > tpnetprotoc.StreamMaxMsgSize {
		stream.Reset()
		err := fmt.Errorf("received message exceeded permissible message maxSize FromPeerID=%s, ProtocolID=%s, ModuleName=%s, size=%d, maxSize=%d",
			streamMsg.FromPeerID,
			streamMsg.ProtocolID,
			streamMsg.ModuleName,
			streamMsg.Size(),
			tpnetprotoc.StreamMaxMsgSize)
		ps.log.Error(err.Error())
		return nil, err
	}

	return &streamMsg, nil
}
