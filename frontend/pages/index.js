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
  const res = await fetch(`http://127.0.0.1:80/api/counter`)
  const data = await res.json()
  return {
    props: {
      HOSTNAME: process.env.HOSTNAME || "",
      counter: data.counter
    }
  }
}
