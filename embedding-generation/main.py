from flask import Flask, request, jsonify
from kafka import KafkaConsumer, KafkaProducer
from flask_cors import CORS, cross_origin
from cross_encoded_embeddings import construct_context, process_data, load_embeddings, create_embeddings
import os
from dotenv import load_dotenv
import json
import pandas as pd
import boto3
from threading import Thread

# Initialize Flask application
app = Flask(__name__)
load_dotenv()
consumer = KafkaConsumer('EMBEDDING_JOBS', bootstrap_servers=['broker:29092'])
producer = KafkaProducer(bootstrap_servers=['broker:29092'], value_serializer=lambda v: json.dumps(v).encode('utf-8'))

# Load the data
# print("loading dataframe...")
# app.df = process_data("output.csv")
# print("loading embeddings...")
# app.embeddings = load_embeddings("embeddings.pkl", "output.csv")

# enable cors
cors = CORS(app)
app.config['CORS_HEADERS'] = 'Content-Type'

@app.route('/context', methods=['POST'])
@cross_origin(origin='*')
def query():
    # Get the input text from the request body
    print(request.json)
    query = request.json['query']
    uuid = request.json['uuid']
    embeddings = load_embeddings(f"{uuid}.pkl")
    # print(embeddings)
    if embeddings == None:
        return jsonify({"error":"embeddings not created yet."})
    dataframe = pd.read_csv(f"cache/{uuid}.csv")
    context = construct_context(query, embeddings, dataframe)
    response_dto = {}
    response_dto['context'] = context
    print(response_dto)
    # send response object to kafka

    return jsonify(response_dto)


def consume():
    aws_access_key_id = os.getenv('AWS_ACCESS_KEY_ID')
    aws_secret_access_key = os.getenv('AWS_SECRET_ACCESS_KEY')
    print("listening for messages...")
    s3_client = boto3.client('s3',
                         aws_access_key_id=aws_access_key_id,
                         aws_secret_access_key=aws_secret_access_key)
    for msg in consumer:
        print("message received!")
        # process message here
        uuid = msg.value.decode()
        saveFileName = f"cache/{uuid}.csv"
        print(saveFileName)
        with open(f"cache/{uuid}.csv", 'wb') as f:
            s3_client.download_fileobj("media-search-engine-bucket", f"transcripts/{uuid}.csv", f)
        create_embeddings(f"{uuid}.pkl", f"cache/{uuid}.csv")
        producer.send("FINISHED_JOBS",uuid)


if __name__ == '__main__':
    Thread(target=consume).start()
    app.run(debug=False, host='0.0.0.0', port='8090') 