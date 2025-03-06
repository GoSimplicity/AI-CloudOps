"""
MIT License

Copyright (c) 2024 Bamboo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

description: 助手路由
"""

from fastapi import APIRouter, HTTPException, BackgroundTasks, Depends, Body, Query
from typing import List, Dict, Any, Optional
from pydantic import BaseModel

from rag.qa_assistant import QaAssistant
from rag.knowledge_base import KnowledgeBase

router = APIRouter()

qa_assistant = None

# 初始化模型
def get_qa_assistant():
    global qa_assistant
    if qa_assistant is None:
        # 创建知识库
        kb = KnowledgeBase(
            docs_dir="./knowledge_docs",
            persist_directory="./data/storage/vector_store" 
        )
        
        # 创建QA助手
        qa_assistant = QaAssistant(knowledge_base=kb)
        
    return qa_assistant

# 请求模型
class QuestionRequest(BaseModel):
    question: str
    chat_history: Optional[List[Dict[str, str]]] = None
    
# 响应模型
class AnswerResponse(BaseModel):
    answer: str
    sources: List[str]
    
# API端点：回答问题
@router.post("/question", response_model=AnswerResponse)
async def answer_question(
    request: QuestionRequest,
    qa: QaAssistant = Depends(get_qa_assistant)
):
    try:
        result = qa.answer_question(
            question=request.question,
            chat_history=request.chat_history
        )
        
        return {
            "answer": result["answer"],
            "sources": result.get("sources", [])
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error answering question: {str(e)}")

# API端点：加载知识库
@router.post("/load-knowledge")
async def load_knowledge(
    background_tasks: BackgroundTasks,
    qa: QaAssistant = Depends(get_qa_assistant)
):
    try:
        # 在后台加载知识库
        background_tasks.add_task(qa.knowledge_base.load_documents)
        return {"message": "Knowledge base loading started in background"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error loading knowledge base: {str(e)}")

# API端点：添加文档
class DocumentItem(BaseModel):
    content: str
    metadata: Dict[str, Any] = {}

@router.post("/add-documents")
async def add_documents(
    documents: List[DocumentItem],
    qa: QaAssistant = Depends(get_qa_assistant)
):
    try:
        docs_list = [{"content": doc.content, "metadata": doc.metadata} for doc in documents]
        qa.knowledge_base.add_documents(docs_list)
        return {"message": f"Added {len(documents)} documents to knowledge base"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error adding documents: {str(e)}")