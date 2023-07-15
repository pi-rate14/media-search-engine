import pandas as pd
import os
import pickle
from sentence_transformers import SentenceTransformer, CrossEncoder, util
from transformers import GPT2TokenizerFast

#We use the Bi-Encoder to encode all passages, so that we can use it with sematic search
bi_encoder = SentenceTransformer('multi-qa-MiniLM-L6-cos-v1')
bi_encoder.max_seq_length = 480     #Truncate long passages to 256 tokens
top_k = 32  

#The bi-encoder will retrieve 100 documents. We use a cross-encoder, to re-rank the results list to improve the quality
cross_encoder = CrossEncoder('cross-encoder/ms-marco-MiniLM-L-6-v2')

def get_embedding(text: str) -> list:
    response = bi_encoder.encode(text, show_progress_bar=True, convert_to_tensor=True)
    return response

def compute_doc_embeddings(df: pd.DataFrame) -> dict:
    """
    Create an embedding for each row in the dataframe using the Sentence Transformers model.
    
    Return a dictionary that maps between each embedding vector and the index of the row that it corresponds to.
    """
    return {
        idx: get_embedding(r.text) for idx, r in df.iterrows()
    }

def process_data(datadoc):

    tokenizer = GPT2TokenizerFast.from_pretrained("gpt2")

    def count_tokens(text: str):
        """count the number of tokens in a string"""
        return len(tokenizer.encode(text))

    def change_values(row):
        row['tokens'] = count_tokens(row['text'])
        return row

    # apply the function to every row in the DataFrame

    df = pd.read_csv(datadoc)
    df = df.apply(change_values, axis=1)
    df.pop(df.columns[0])
    df.to_csv(datadoc)
    return df

def load_embeddings(fname: str, datadoc:str=None):
    """
    Read the document embeddings and their keys from a CSV.
    """
    if not os.path.exists(f"embeddings/{fname}"):
        # read your corpus etc
        return None
    else:
        print("Loading pre-computed embeddings from disc")
        with open(f"embeddings/{fname}", "rb") as fIn:
            stored_data = pickle.load(fIn)
            document_embeddings = stored_data['embeddings']
    
    return document_embeddings

def create_embeddings(fname: str, datadoc:str):
    print("Encoding the corpus. This might take a while")
    df = process_data(datadoc)

    document_embeddings = compute_doc_embeddings(df)

    print("Storing file on disc")
    with open(f"embeddings/{fname}", "wb") as fOut:
        pickle.dump({'embeddings': document_embeddings}, fOut, protocol=pickle.HIGHEST_PROTOCOL)

def get_relevant_documents(query, contexts):
    """
    Find the query embedding for the supplied query, and compare it against all of the pre-calculated document embeddings
    to find the most relevant sections. 
    Return the list of document sections, sorted by relevance in descending order.
    """
    query_embedding = get_embedding(query)
    doc_embeddings = [embeddings for idx,embeddings in contexts.items()]
    hits = util.semantic_search(query_embedding, doc_embeddings, top_k=top_k)
    hits = hits[0]
   

    return hits

def order_document_sections_by_query_similarity(query, hits, df):
    """
    Find the query embedding for the supplied query, and compare it against all of the pre-calculated document embeddings
    to find the most relevant sections. 
    Return the list of document sections, sorted by relevance in descending order.
    """

    cross_inp = [[query, df.loc[hit['corpus_id']].text] for hit in hits]
    cross_scores = cross_encoder.predict(cross_inp)

      # Sort results by the cross-encoder scores
    for idx in range(len(cross_scores)):
        hits[idx]['cross-score'] = cross_scores[idx]

    hits = sorted(hits, key=lambda x: x['cross-score'], reverse=True)
    return hits

def construct_context(question: str, context_embeddings: dict, df: pd.DataFrame):
    MAX_SECTION_LEN = 250
    SEPARATOR = "###"
    """
    Fetch relevant documents and concatenate them together to create a context
    """
    relevant_document_sections = get_relevant_documents(question, context_embeddings)
    most_relevant_document_sections = order_document_sections_by_query_similarity(question, relevant_document_sections, df)
    chosen_sections = []
    chosen_sections_len = 0
    chosen_sections_indexes = []
     
    for each in most_relevant_document_sections:
        # Add contexts until we run out of space.
        section_index = each['corpus_id']      
        document_section = df.loc[section_index]
        chosen_sections_len += document_section.tokens
        if chosen_sections_len > MAX_SECTION_LEN:
            break
            
        chosen_sections.append(SEPARATOR + document_section.text.replace("\n", " "))
        chosen_sections_indexes.append(str(section_index))

    final_context = ""
    for i in range(len(chosen_sections)-1):
        final_context += chosen_sections[i]
    return final_context

