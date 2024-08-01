const userForm = document.getElementById("userForm");

userForm.onsubmit = e => {
	e.preventDefault();

	const fd = new FormData(userForm);

	const params = new URLSearchParams(window.location.search);

	fd.set("invite", params.get("c"));

	fetch("/api/register", {
		method: "POST",
		body: fd
	});
}
