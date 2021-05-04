package bus

import (
	"sync"

	"github.com/golang/glog"
)

type AgentQuestionChan chan AgentQuestion
type AgentAnswerChan chan AgentAnswer

type AgentQuestion struct {
	AgentNotify
	AgentAnswerChan
}

type AgentAnswer struct {
	ID QuestionKeyType
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

	glog.V(3).Infoln(key.AgentDID, " answerer REMOVE for:", key.ClientID)
	delete(questionChannels[m].agentQuestionMap, key)
}

// AgentSendQuestion sends to question to first found answerer and returns a
// channel for the answer. If answerer doesn't exist it returns nil.
func (m mapIndex) AgentSendQuestion(question AgentQuestion) AgentAnswerChan {
	questionChannels[m].Lock() // cannot use defer for unlocking, see below

	key := question.AgentKeyType
	// send question to first answerer
	for k, ch := range questionChannels[m].agentQuestionMap {
		if key.AgentDID == k.AgentDID {
			questionChannels[m].Unlock() // first safe opportunity

			glog.V(3).Infoln(key.AgentDID, " agent QUESTION ID:", question.ID)
			question.ClientID = k.ClientID
			question.AgentAnswerChan = make(AgentAnswerChan, 1)
			askedQuestions[m].Lock()
			askedQuestions[m].questionMap[question.ID] = question
			askedQuestions[m].Unlock()
			ch <- question
			return question.AgentAnswerChan
		}
	}
	questionChannels[m].Unlock() // second exit point
	return nil
}

// AgentSendAnswer is the function where answer to a question can be send by
// clientID which must be registered AgentAddAnswerer
func (m mapIndex) AgentSendAnswer(answer AgentAnswer) {
	askedQuestions[m].Lock() // cannot use defer unlocking, see below

	if q, ok := askedQuestions[m].questionMap[answer.ID]; ok {
		c := q.AgentAnswerChan
		delete(askedQuestions[m].questionMap, answer.ID)
		askedQuestions[m].Unlock()
		glog.V(3).Infoln(q.AgentDID, " agent ANSWER for QID:", answer.ID)
		c <- answer
	} else {
		askedQuestions[m].Unlock()
		glog.Warningf("couldn't find question channel for %s", answer.ID)
	}
}
