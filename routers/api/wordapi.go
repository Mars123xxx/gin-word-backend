package api

import "github.com/gin-gonic/gin"

func wordIndex(c *gin.Context) {
	c.String(200, "wordapi is ok")
}

func GetWordBaseDetailByID(id int) map[string]interface{} {
	baseDetail := make(map[string]interface{})
	return baseDetail
}

//def getWordBaseDetailById(_id: int) -> dict:
//word = Word.query.get(_id)
//sentences = Sentence.query.filter_by(word_id=word.id).all()
//
//s_list = [{
//'content': s.content,
//'from_word_id': word.id
//} for s in sentences]
//meanings = Meaning.query.filter_by(word_id=word.id).all()
//
//m_list = [{
//'part_of_speech': m.part_of_speech,
//'definition': m.definition,
//'from_word_id': word.id
//} for m in meanings]
//collocations = Collocation.query.filter_by(word_id=word.id).all()
//
//c_list = [{
//'content': c.content,
//'from_word_id': word.id
//} for c in collocations]
//relatives = Relative.query.filter_by(word_id=word.id).all()
//
//r_list = [{
//'content': r.content,
//'from_word_id': word.id
//} for r in relatives]
//
//word_detail = {
//'id': word.id,
//'word': word.word,
//'language': word.language,
//'sentences': s_list,
//'root_word': word.root_word,
//'root_meaning': word.root_meaning,
//'meanings': m_list,
//'collocations': c_list,
//'relative_words': r_list
//}

func SetupWordRouter(wordGroup *gin.RouterGroup) {
	wordGroup.GET("/", wordIndex)
}
