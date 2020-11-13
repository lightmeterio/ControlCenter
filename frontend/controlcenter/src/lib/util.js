export function togglePasswordShow() {
  let attrValue = document
    .querySelector("#show_hide_password input")
    .getAttribute("type");
  if (attrValue === "text") {
    document
      .querySelector("#show_hide_password input")
      .setAttribute("type", "password");
    let i = document.querySelector("#show_hide_password svg");
    i.classList.add("fa-eye");
    i.classList.remove("fa-eye-slash");
  } else if (attrValue === "password") {
    document
      .querySelector("#show_hide_password input")
      .setAttribute("type", "text");
    let i = document.querySelector("#show_hide_password svg");
    i.classList.remove("fa-eye");
    i.classList.add("fa-eye-slash");
  }
}
