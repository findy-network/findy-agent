package bus

import (
	"sync"

	"github.com/golang/glog"
)

type AgentQuestionChan chan AgentQuestion
type AgentAnswerChan chan AgentAnswer

type AgentQuestion struct {
	QID QuestionKeyType
	AgentNotify
	AgentAnswerChan
}

type AgentAnswer struct {
	QID QuestionKeyType
	AgentKeyType

	ACK  bool
	Info string
}

const (
	agentToAsk = 0 + iota
)

type agentQuestionMap map[AgentKeyType]AgentQuestionChan
type agentQuestionLockMap struct {
	agentQuestionMap
	sync.Mutex
}

type QuestionKeyType = string

type questionMap map[QuestionKeyType]AgentQuestion
type questionLMap struct {
	questionMap
	sync.Mutex
}

var questionChannels = [...]agentQuestionLockMap{{agentQuestionMap: make(agentQuestionMap)}}
var askedQuestions = [...]questionLMap{{questionMap: make(questionMap)}}

var WantAllAgentAnswers mapIndex = agentToAsk

// AgentAddAnswerer adds the answerer and returns the question channel to listen
// for.
func (m mapIndex) AgentAddAnswerer(key AgentKeyType) AgentQuestionChan {
	questionChannels[m].Lock()
	defer questionChannels[m].Unlock()

	glog.V(3).Infoln(key.AgentDID, " answerer ADD for:", key.ClientID)
	questionChannels[m].agentQuestionMap[key] = make(AgentQuestionChan, 1)
	return questionChannels[m].agentQuestionMap[key]
}

// AgentRmAnswerer removes answerer.
func (m mapIndex) AgentRmAnswerer(key AgentKeyType) {
	questionChannels[m].Lock()
	defer questionChannels[m].Unlock()

	glog.V(3).Infoln(key.AgentDID, " answerer RM for:", key.ClientID)
	delete(questionChannels[m].agentQuestionMap, key)
}

// AgentSendQuestion sends to question to first found answerer and returns a
// channel for the answer. If answerer doesn't exist it returns nil.
func (m mapIndex) AgentSendQuestion(question AgentQuestion) AgentAnswerChan {
	questionChannels[m].Lock()
	defer questionChannels[m].Unlock()

	key := question.AgentKeyType
	// send question to first answerer
	for k, ch := range questionChannels[m].agentQuestionMap {
		if key.AgentDID == k.AgentDID {
			glog.V(3).Infoln(key.AgentDID, " agent QUESTION:", k.ClientID)
			question.AgentKeyType.ClientID = k.ClientID
			question.AgentAnswerChan = make(AgentAnswerChan, 1)
			askedQuestions[m].questionMap[question.QID] = question
			ch <- question
			return question.AgentAnswerChan
		}
	}
	return nil
}

// AgentSendAnswer is the function where answer to a question can be send by
// clientID which must be registered AgentAddAnswerer
func (m mapIndex) AgentSendAnswer(answer AgentAnswer) {
	askedQuestions[m].Lock()
	defer askedQuestions[m].Unlock()

	if q, ok := askedQuestions[m].questionMap[answer.QID]; ok {
		glog.V(3).Infoln(q.AgentDID, " agent ANSWER:", q.ClientID)
		q.AgentAnswerChan <- answer
	}
}
