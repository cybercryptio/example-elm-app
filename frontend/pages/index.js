import React from "react";

export default function Index(props) {
  return (
    <body style={{
      height: "100%",
    }}>
    <section  style={{
        width: "100%",
        height: "100%",
        display: "table",
        textAlign: "center",
    }}>
		<div style={{
          display: "table-cell",
          verticalAlign: "middle",
    }}>
    <h1 style={{
          textAlign: "center",
          fontSize: "5rem",
        }}>👋 {props.HOSTNAME} {props.counter}</h1>
		</div>
	</section>
  </body>
  );
}

export async function getServerSideProps() {
  var API_ORIGIN = process.env.API_ORIGIN || "http://127.0.0.1"
  const res = await fetch(API_ORIGIN+`/api/counter`)
  const data = await res.json()
  return {
    props: {
      HOSTNAME: process.env.HOSTNAME || "",
      counter: data.counter
    }
  }
}
