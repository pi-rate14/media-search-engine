import { useState, useEffect } from "react";
import axios from "axios";
import "./App.css";

function App() {
  const [file, setFile] = useState(null);
  const [link, setLink] = useState("");
  const [pollingUrl, setPollingUrl] = useState("");
  const [jobID, setJobId] = useState("");
  const [pollingStatus, setPollingStatus] = useState("PENDING");
  const [fileName, setFileName] = useState("");
  const [question, setQuestion] = useState("");
  const [loading, setLoading] = useState(false);
  const [answer, setAnswer] = useState("");

  const extractYouTubeVideoId = (url) => {
    const match = url.match(
      /^(?:https?:\/\/)?(?:www\.)?youtube\.com\/watch\?v=([^&]+)/i
    );
    return match && match[1];
  };

  const getEmbeddedYouTubeUrl = () => {
    const videoId = extractYouTubeVideoId(link);
    if (videoId) {
      return `https://www.youtube.com/embed/${videoId}`;
    }
    return null;
  };

  const handleFileUpload = (event) => {
    setFileName(event.target.files[0].name);
    setFile(event.target.files[0]);
  };

  const removeFile = () => {
    setFileName("");
    setLink("");
    setFile(null);
  };

  const handleLinkInput = (event) => {
    setLink(event.target.value);
    setFileName(event.target.value);
  };

  const handleQuestion = (event) => {
    setQuestion(event.target.value);
  };

  const handleFormSubmit = async (event) => {
    event.preventDefault();
    setLoading(true);
    if (file) {
      let formData = new FormData();
      formData.append("video", file);
      try {
        const response = await axios.post(
          "http://localhost:8085/create",
          formData
        );
        if (response.status === 201) {
          const pollingUrl = response.data.poll_url;
          const uuid = response.data.uuid;
          setJobId(uuid);
          setPollingUrl(pollingUrl);
        } else {
          alert("ISE");
        }
      } catch (error) {
        // Handle error case
        alert(error);
      }
    } else if (link) {
      const payload = {};
      payload.link = link;
      try {
        const response = await axios.post(
          "http://localhost:8085/createFromLink",
          payload
        );
        if (response.status === 201) {
          const pollingUrl = response.data.poll_url;
          const uuid = response.data.uuid;
          console.log(pollingUrl);
          setJobId(uuid);
          setPollingUrl(pollingUrl);
        } else {
          alert("ISE");
        }
      } catch (error) {
        // Handle error case
        alert(error);
      }
    }
    setLoading(false);
  };

  const pollStatus = async () => {
    setLoading(true);
    try {
      const response = await axios.get(pollingUrl);
      if (response.status === 200) {
        console.log(response.data.status);
        if (response.data.status === "COMPLETED") {
          setPollingStatus("COMPLETED");
          setLoading(false);
        } else if (response.data.status === "FAILED") {
          setLoading(false);
          return;
        } else {
          // Handle error case
          setTimeout(pollStatus, 5000);
        }
      }
    } catch (error) {
      // Handle error case
    }
  };

  const queryQuestion = async () => {
    setLoading(true);
    try {
      const payload = {};
      payload.query = question;
      payload.uuid = jobID;
      const response = await axios.post("http://localhost:8085/query", payload);
      if (response.status === 200) {
        const { data } = response.data;
        setAnswer(data);
      } else {
        // Handle error case
      }
    } catch (error) {
      // Handle error case
      alert(error);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (pollingUrl && pollingStatus === "PENDING") {
      pollStatus();
    }
  }, [pollingUrl, pollingStatus]);

  return (
    <>
      <div
        style={{
          width: "80%",
          margin: "auto",
          padding: "10%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <div style={{"width":"100%"}}>
          <input type="file" onChange={handleFileUpload} />
          <br />
          <h2>OR</h2>
          <label>Paste YouTube Link</label>
          <br />
          <input
            type="text"
            disabled={file ? true : false}
            value={fileName}
            onChange={handleLinkInput}
          />
          {link && (
            <>
              <div>
                <iframe
                  width="300"
                  height="200"
                  src="https://www.youtube.com/embed/J7GY1Xg6X20"
                  title="YouTube video player"
                  frameborder="0"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                  allowfullscreen
                ></iframe>
              </div>
            </>
          )}
          {file && (
            <>
              <div>
                <video controls style={{ width: "90%", height: "80%" }}>
                  <source src={URL.createObjectURL(file)} type="video/mp4" />
                  Your browser does not support the video tag.
                </video>
              </div>

              <button type="button" onClick={removeFile}>
                Clear
              </button>
            </>
          )}
          <button type="button" onClick={handleFormSubmit}>
            Go
          </button>
        </div>
      </div>
      <div
        style={{
          width: "80%",
          margin: "auto",
          padding: "10%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        {pollingStatus === "COMPLETED" && (
          <div style={{ width: "100%", margin: "auto" }}>
            <br />
            <h3>Question:</h3>
            <input type="text" value={question} onChange={handleQuestion} />
            <br />
            <br />
            <button type="button" onClick={queryQuestion}>
              ASK
            </button>
            <h3>Answer:</h3>
            {answer && (
              <p style={{ wordWrap: "break-word", lineHeight: "1.5" }} dangerouslySetInnerHTML={{ __html: answer }}>
               
              </p>
            )}
          </div>
        )}
      </div>
      {loading && <h1>PROCESSING...</h1>}
    </>
  );
}

export default App;
